package postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
	"unicode/utf16"

	"github.com/example/medusa-go-clone/internal/modules/locking/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const Identifier = "locking-postgres"

// Provider ports Medusa's PostgreSQL locking provider. Execute uses session
// advisory locks so the lock remains held while the supplied application job
// performs its own repository calls. Explicit acquire/release use the locking
// table, matching the source provider's owner and expiration records.
type Provider struct{ pool *pgxpool.Pool }

func NewProvider(pool *pgxpool.Pool) *Provider { return &Provider{pool: pool} }
func (*Provider) Identifier() string           { return Identifier }

func (p *Provider) Execute(ctx context.Context, keys []string, timeout time.Duration, job func(context.Context) error) error {
	if p.pool == nil {
		return errors.New("locking-postgres requires a PostgreSQL pool")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := p.pool.Acquire(lockCtx)
	if err != nil {
		return err
	}
	defer conn.Release()
	lockIDs := advisoryKeys(keys)
	acquired := make([]int64, 0, len(lockIDs))
	defer func() {
		for i := len(acquired) - 1; i >= 0; i-- {
			_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", acquired[i])
		}
	}()
	for _, id := range lockIDs {
		if _, err = conn.Exec(lockCtx, "SELECT pg_advisory_lock($1)", id); err != nil {
			return err
		}
		acquired = append(acquired, id)
	}
	return job(lockCtx)
}

func (p *Provider) Acquire(ctx context.Context, keys []string, options domain.LockOptions) error {
	for _, key := range keys {
		var owner *string
		var expiration *time.Time
		var now time.Time
		err := p.pool.QueryRow(ctx, "SELECT owner_id,expiration,NOW() FROM locking WHERE id=$1", key).Scan(&owner, &expiration, &now)
		if errors.Is(err, pgx.ErrNoRows) {
			var exp any
			if options.Expire > 0 {
				exp = options.Expire.String()
			}
			_, err = p.pool.Exec(ctx, "INSERT INTO locking(id,owner_id,expiration) VALUES($1,$2,CASE WHEN $3::text IS NULL THEN NULL ELSE NOW()+$3::interval END)", key, nullableOwner(options.OwnerID), exp)
			if err != nil {
				// A concurrent insert is only accepted for the same owner, as in the source adapter.
				var current *string
				if readErr := p.pool.QueryRow(ctx, "SELECT owner_id FROM locking WHERE id=$1", key).Scan(&current); readErr != nil {
					return err
				}
				if !sameOwner(current, options.OwnerID) {
					return fmt.Errorf("%q is already locked", key)
				}
				continue
			}
			continue
		}
		if err != nil {
			return err
		}
		if owner == nil || *owner != options.OwnerID {
			return fmt.Errorf("failed to acquire lock for key %q", key)
		}
		if expiration == nil {
			continue
		}
		if options.Expire > 0 && !expiration.After(now) {
			if _, err := p.pool.Exec(ctx, "UPDATE locking SET expiration=NOW()+$1::interval WHERE id=$2", options.Expire.String(), key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Provider) Release(ctx context.Context, keys []string, ownerID string) (bool, error) {
	success := true
	for _, key := range keys {
		var owner *string
		var expiration *time.Time
		var now time.Time
		err := p.pool.QueryRow(ctx, "SELECT owner_id,expiration,NOW() FROM locking WHERE id=$1", key).Scan(&owner, &expiration, &now)
		if errors.Is(err, pgx.ErrNoRows) {
			success = false
			continue
		}
		if err != nil {
			return false, err
		}
		if !sameOwner(owner, ownerID) {
			success = false
			continue
		}
		if _, err := p.pool.Exec(ctx, "DELETE FROM locking WHERE id=$1", key); err != nil {
			return false, err
		}
		success = success && (expiration == nil || expiration.After(now))
	}
	return success, nil
}

func (p *Provider) ReleaseAll(ctx context.Context, ownerID *string) error {
	if ownerID == nil {
		_, err := p.pool.Exec(ctx, "TRUNCATE TABLE locking")
		return err
	}
	_, err := p.pool.Exec(ctx, "DELETE FROM locking WHERE owner_id=$1", *ownerID)
	return err
}

func nullableOwner(owner string) any {
	if owner == "" {
		return nil
	}
	return owner
}
func sameOwner(owner *string, expected string) bool {
	if expected == "" {
		return owner == nil
	}
	return owner != nil && *owner == expected
}

func advisoryKeys(keys []string) []int64 {
	seen := map[int64]bool{}
	ids := make([]int64, 0, len(keys))
	for _, key := range keys {
		// Medusa hashes UTF-16 code units from right to left with the djb2
		// recurrence and stores the unsigned 32-bit result.
		units := utf16.Encode([]rune(key))
		hash := uint32(5381)
		for i := len(units) - 1; i >= 0; i-- {
			hash = hash*33 ^ uint32(units[i])
		}
		id := int64(hash)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

var _ domain.Provider = (*Provider)(nil)
