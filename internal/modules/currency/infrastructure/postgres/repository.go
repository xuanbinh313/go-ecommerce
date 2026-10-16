package postgres

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/currency/domain"
	"github.com/example/medusa-go-clone/internal/modules/currency/domain/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func (r *Repository) Retrieve(ctx context.Context, code string) (models.Currency, error) {
	var c models.Currency
	err := r.db.QueryRow(ctx, `SELECT code,symbol,symbol_native,name,decimal_digits,rounding::text,created_at,updated_at FROM currency WHERE code=$1`, code).Scan(&c.Code, &c.Symbol, &c.SymbolNative, &c.Name, &c.DecimalDigits, &c.Rounding, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}
func (r *Repository) List(ctx context.Context, codes []string) ([]models.Currency, error) {
	query := `SELECT code,symbol,symbol_native,name,decimal_digits,rounding::text,created_at,updated_at FROM currency`
	args := []any{}
	if len(codes) > 0 {
		query += ` WHERE code=ANY($1)`
		args = append(args, codes)
	}
	query += ` ORDER BY code`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Currency{}
	for rows.Next() {
		var c models.Currency
		if err := rows.Scan(&c.Code, &c.Symbol, &c.SymbolNative, &c.Name, &c.DecimalDigits, &c.Rounding, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Repository) Upsert(ctx context.Context, currencies []models.Currency) ([]models.Currency, error) {
	for _, c := range currencies {
		if _, err := r.db.Exec(ctx, `INSERT INTO currency(code,symbol,symbol_native,name,decimal_digits,rounding) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(code) DO UPDATE SET symbol=EXCLUDED.symbol,symbol_native=EXCLUDED.symbol_native,name=EXCLUDED.name,decimal_digits=EXCLUDED.decimal_digits,rounding=EXCLUDED.rounding,updated_at=now()`, c.Code, c.Symbol, c.SymbolNative, c.Name, c.DecimalDigits, c.Rounding); err != nil {
			return nil, err
		}
	}
	codes := make([]string, len(currencies))
	for i, c := range currencies {
		codes[i] = c.Code
	}
	return r.List(ctx, codes)
}

var _ domain.Repository = (*Repository)(nil)
