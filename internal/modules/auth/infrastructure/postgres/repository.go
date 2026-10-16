package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain"
	"github.com/example/medusa-go-clone/internal/modules/auth/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) CreateIdentityWithProvider(ctx context.Context, provider, email, passwordHash string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	identityID, providerID := entityid.New("authid"), entityid.New("")
	metadata, err := json.Marshal(map[string]string{"password": passwordHash})
	if err != nil {
		return "", err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO auth_identity(id) VALUES($1)`, identityID); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO provider_identity(id,auth_identity_id,provider,entity_id,provider_metadata) VALUES($1,$2,$3,$4,$5)`, providerID, identityID, provider, email, metadata); err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return identityID, nil
}

func (r *Repository) DeleteIdentity(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM auth_identity WHERE id=$1`, id)
	return err
}

func (r *Repository) FindIdentityByActor(ctx context.Context, actorType, actorID string) (domain.AuthIdentityRecord, error) {
	var identity domain.AuthIdentityRecord
	err := r.db.QueryRow(ctx, `SELECT id,app_metadata FROM auth_identity WHERE app_metadata->>$1=$2 AND deleted_at IS NULL ORDER BY created_at LIMIT 1`, actorType+"_id", actorID).Scan(&identity.ID, &identity.AppMetadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthIdentityRecord{}, domain.ErrIdentityNotFound
	}
	return identity, err
}

func (r *Repository) RetrieveAuthIdentity(ctx context.Context, identityID, provider string) (domain.AuthIdentitySnapshot, error) {
	var identity domain.AuthIdentitySnapshot
	err := r.db.QueryRow(ctx, `SELECT a.id,COALESCE(a.app_metadata,'{}'::jsonb),COALESCE(p.user_metadata,'{}'::jsonb) FROM auth_identity a JOIN provider_identity p ON p.auth_identity_id=a.id AND p.provider=$2 WHERE a.id=$1 AND a.deleted_at IS NULL ORDER BY p.created_at LIMIT 1`, identityID, provider).Scan(&identity.ID, &identity.AppMetadata, &identity.UserMetadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthIdentitySnapshot{}, domain.ErrIdentityNotFound
	}
	return identity, err
}

func (r *Repository) FindOAuthIdentity(ctx context.Context, provider, entityID string) (domain.OAuthIdentity, error) {
	var identity domain.OAuthIdentity
	err := r.db.QueryRow(ctx, `SELECT a.id,COALESCE(a.app_metadata,'{}'::jsonb),COALESCE(p.user_metadata,'{}'::jsonb) FROM auth_identity a JOIN provider_identity p ON p.auth_identity_id=a.id AND p.provider=$1 WHERE p.entity_id=$2 AND a.deleted_at IS NULL AND p.deleted_at IS NULL ORDER BY p.created_at LIMIT 1`, provider, entityID).Scan(&identity.IdentityID, &identity.AppMetadata, &identity.UserMetadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OAuthIdentity{}, domain.ErrIdentityNotFound
	}
	return identity, err
}

func (r *Repository) CreateOAuthIdentity(ctx context.Context, provider, entityID string, providerMetadata, userMetadata json.RawMessage) (domain.OAuthIdentity, error) {
	identity := domain.OAuthIdentity{IdentityID: entityid.New("authid"), AppMetadata: json.RawMessage(`{}`), UserMetadata: userMetadata}
	if len(providerMetadata) == 0 {
		providerMetadata = json.RawMessage(`{}`)
	}
	if len(userMetadata) == 0 {
		userMetadata = json.RawMessage(`{}`)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.OAuthIdentity{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO auth_identity(id) VALUES($1)`, identity.IdentityID); err != nil {
		return domain.OAuthIdentity{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO provider_identity(id,auth_identity_id,provider,entity_id,provider_metadata,user_metadata) VALUES($1,$2,$3,$4,$5,$6)`, entityid.New(""), identity.IdentityID, provider, entityID, providerMetadata, userMetadata); err != nil {
		return domain.OAuthIdentity{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.OAuthIdentity{}, err
	}
	return identity, nil
}

func (r *Repository) UpdateOAuthIdentity(ctx context.Context, provider, entityID string, providerMetadata, userMetadata json.RawMessage) (domain.OAuthIdentity, error) {
	var identity domain.OAuthIdentity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return identity, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `SELECT a.id,COALESCE(a.app_metadata,'{}'::jsonb) FROM auth_identity a JOIN provider_identity p ON p.auth_identity_id=a.id WHERE p.provider=$1 AND p.entity_id=$2 AND a.deleted_at IS NULL AND p.deleted_at IS NULL ORDER BY p.created_at LIMIT 1 FOR UPDATE OF p`, provider, entityID).Scan(&identity.IdentityID, &identity.AppMetadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OAuthIdentity{}, domain.ErrIdentityNotFound
	}
	if err != nil {
		return identity, err
	}
	if len(providerMetadata) == 0 {
		providerMetadata = json.RawMessage(`{}`)
	}
	if len(userMetadata) == 0 {
		userMetadata = json.RawMessage(`{}`)
	}
	if _, err = tx.Exec(ctx, `UPDATE provider_identity SET provider_metadata=$3,user_metadata=$4,updated_at=now() WHERE provider=$1 AND entity_id=$2 AND deleted_at IS NULL`, provider, entityID, providerMetadata, userMetadata); err != nil {
		return identity, err
	}
	if err = tx.Commit(ctx); err != nil {
		return identity, err
	}
	identity.UserMetadata = userMetadata
	return identity, nil
}

func (r *Repository) UpdateOAuthUserMetadata(ctx context.Context, provider, entityID string, userMetadata json.RawMessage) (domain.OAuthIdentity, error) {
	var identity domain.OAuthIdentity
	err := r.db.QueryRow(ctx, `UPDATE provider_identity p SET user_metadata=$3,updated_at=now() FROM auth_identity a WHERE p.auth_identity_id=a.id AND p.provider=$1 AND p.entity_id=$2 AND p.deleted_at IS NULL AND a.deleted_at IS NULL RETURNING a.id,COALESCE(a.app_metadata,'{}'::jsonb),COALESCE(p.user_metadata,'{}'::jsonb)`, provider, entityID, userMetadata).Scan(&identity.IdentityID, &identity.AppMetadata, &identity.UserMetadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OAuthIdentity{}, domain.ErrIdentityNotFound
	}
	return identity, err
}

func (r *Repository) ListProviderNamesByActor(ctx context.Context, actorType, actorID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT p.provider FROM auth_identity a JOIN provider_identity p ON p.auth_identity_id=a.id WHERE a.app_metadata->>$1=$2 AND a.deleted_at IS NULL`, actorType+"_id", actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	providers := []string{}
	for rows.Next() {
		var provider string
		if err = rows.Scan(&provider); err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (r *Repository) ListProviderNamesByIdentity(ctx context.Context, identityID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT provider FROM provider_identity WHERE auth_identity_id=$1 AND deleted_at IS NULL ORDER BY created_at`, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	providers := []string{}
	for rows.Next() {
		var provider string
		if err = rows.Scan(&provider); err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, provider, entityID string, ttl time.Duration) (string, time.Time, error) {
	if ttl <= 0 {
		return "", time.Time{}, errors.New("Password reset token TTL must be a positive integer")
	}
	var providerIdentityID, identityID string
	if err := r.db.QueryRow(ctx, `SELECT id,auth_identity_id FROM provider_identity WHERE provider=$1 AND entity_id=$2 ORDER BY created_at LIMIT 1`, provider, entityID).Scan(&providerIdentityID, &identityID); err != nil {
		return "", time.Time{}, err
	}
	jti, err := newJTI()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(ttl)
	sum := sha256.Sum256([]byte(jti))
	id := entityid.New("authprt")
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE auth_password_reset_token SET deleted_at=now(),updated_at=now() WHERE provider_identity_id=$1 AND deleted_at IS NULL`, providerIdentityID); err != nil {
		return "", time.Time{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO auth_password_reset_token(id,auth_identity_id,provider_identity_id,entity_id,token_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, id, identityID, providerIdentityID, entityID, hex.EncodeToString(sum[:]), expires); err != nil {
		return "", time.Time{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", time.Time{}, err
	}
	return jti, expires, nil
}

func (r *Repository) ConsumePasswordResetToken(ctx context.Context, jti, provider, entityID string) error {
	sum := sha256.Sum256([]byte(jti))
	hash := hex.EncodeToString(sum[:])
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	var expires time.Time
	var storedProvider, storedEntity string
	err = tx.QueryRow(ctx, `SELECT t.id,t.expires_at,p.provider,p.entity_id FROM auth_password_reset_token t JOIN provider_identity p ON p.id=t.provider_identity_id WHERE t.token_hash=$1 AND t.deleted_at IS NULL FOR UPDATE OF t`, hash).Scan(&id, &expires, &storedProvider, &storedEntity)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidPasswordResetToken
	}
	if err != nil {
		return err
	}
	if !expires.After(time.Now()) || storedProvider != provider || storedEntity != entityID {
		if _, e := tx.Exec(ctx, `UPDATE auth_password_reset_token SET deleted_at=now(),updated_at=now() WHERE id=$1`, id); e != nil {
			return e
		}
		if e := tx.Commit(ctx); e != nil {
			return e
		}
		return domain.ErrInvalidPasswordResetToken
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_password_reset_token SET deleted_at=now(),updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RequestAuthVerification(ctx context.Context, in models.RequestAuthVerificationInput, ttl time.Duration) (models.AuthVerification, error) {
	if ttl <= 0 {
		return models.AuthVerification{}, errors.New("Verification token TTL must be a positive integer")
	}
	if in.AuthIdentityID == "" || in.EntityID == "" || in.EntityType == "" || in.CodeProvider == "" {
		return models.AuthVerification{}, errors.New("auth identity, entity, entity type, and code provider are required")
	}
	now := time.Now().UTC()
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return models.AuthVerification{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(token))
	providerMetadata, err := json.Marshal(map[string]string{"token_hash": hex.EncodeToString(hash[:])})
	if err != nil {
		return models.AuthVerification{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.AuthVerification{}, err
	}
	defer tx.Rollback(ctx)
	var existing models.AuthVerification
	err = tx.QueryRow(ctx, `SELECT id,auth_identity_id,entity_id,entity_type,code_provider,verified_at,requested_at,COALESCE(provider_metadata,'{}'::jsonb),COALESCE(metadata,'{}'::jsonb) FROM auth_verification WHERE auth_identity_id=$1 AND entity_id=$2 AND entity_type=$3 AND deleted_at IS NULL ORDER BY created_at LIMIT 1 FOR UPDATE`, in.AuthIdentityID, in.EntityID, in.EntityType).Scan(&existing.ID, &existing.AuthIdentityID, &existing.EntityID, &existing.EntityType, &existing.CodeProvider, &existing.VerifiedAt, &existing.RequestedAt, &existing.ProviderMetadata, &existing.Metadata)
	if err == nil && existing.VerifiedAt != nil {
		if err = tx.Commit(ctx); err != nil {
			return models.AuthVerification{}, err
		}
		return existing, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return models.AuthVerification{}, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		metadata := in.Metadata
		if len(metadata) == 0 {
			metadata = json.RawMessage("null")
		}
		existing = models.AuthVerification{ID: entityid.New("authver"), AuthIdentityID: in.AuthIdentityID, EntityID: in.EntityID, EntityType: in.EntityType, CodeProvider: in.CodeProvider, RequestedAt: now, ProviderMetadata: providerMetadata, Metadata: metadata}
		if _, err = tx.Exec(ctx, `INSERT INTO auth_verification(id,auth_identity_id,entity_id,entity_type,code_provider,verified_at,requested_at,provider_metadata,metadata) VALUES($1,$2,$3,$4,$5,NULL,$6,$7,$8)`, existing.ID, existing.AuthIdentityID, existing.EntityID, existing.EntityType, existing.CodeProvider, existing.RequestedAt, existing.ProviderMetadata, existing.Metadata); err != nil {
			return models.AuthVerification{}, err
		}
	} else {
		existing.CodeProvider = in.CodeProvider
		existing.VerifiedAt = nil
		existing.RequestedAt = now
		existing.ProviderMetadata = providerMetadata
		if _, err = tx.Exec(ctx, `UPDATE auth_verification SET code_provider=$2,provider_metadata=$3,requested_at=$4,verified_at=NULL,updated_at=now() WHERE id=$1`, existing.ID, existing.CodeProvider, existing.ProviderMetadata, now); err != nil {
			return models.AuthVerification{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.AuthVerification{}, err
	}
	expires := now.Add(ttl)
	existing.Code = token
	existing.ExpiresAt = &expires
	return existing, nil
}

func (r *Repository) ConfirmAuthVerification(ctx context.Context, in models.ConfirmAuthVerificationInput, ttl time.Duration) (models.AuthVerification, error) {
	if in.Code == "" {
		return models.AuthVerification{}, errors.New("Verification code is required")
	}
	if ttl <= 0 {
		return models.AuthVerification{}, errors.New("Verification token TTL must be a positive integer")
	}
	hash := sha256.Sum256([]byte(in.Code))
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.AuthVerification{}, err
	}
	defer tx.Rollback(ctx)
	var verification models.AuthVerification
	err = tx.QueryRow(ctx, `SELECT id,auth_identity_id,entity_id,entity_type,code_provider,verified_at,requested_at,COALESCE(provider_metadata,'{}'::jsonb),COALESCE(metadata,'{}'::jsonb) FROM auth_verification WHERE provider_metadata->>'token_hash'=$1 AND deleted_at IS NULL AND ($2='' OR auth_identity_id=$2) FOR UPDATE`, hex.EncodeToString(hash[:]), in.AuthIdentityID).Scan(&verification.ID, &verification.AuthIdentityID, &verification.EntityID, &verification.EntityType, &verification.CodeProvider, &verification.VerifiedAt, &verification.RequestedAt, &verification.ProviderMetadata, &verification.Metadata)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && verification.VerifiedAt != nil) {
		return models.AuthVerification{}, errors.New("Verification code is invalid or already used")
	}
	if err != nil {
		return models.AuthVerification{}, err
	}
	if in.CodeProvider != "" && in.CodeProvider != verification.CodeProvider {
		return models.AuthVerification{}, fmt.Errorf("Verification code does not belong to provider %q", in.CodeProvider)
	}
	if !verification.RequestedAt.Add(ttl).After(time.Now()) {
		return models.AuthVerification{}, errors.New("Verification code has expired")
	}
	now := time.Now().UTC()
	if _, err = tx.Exec(ctx, `UPDATE auth_verification SET verified_at=$2,updated_at=now() WHERE id=$1`, verification.ID, now); err != nil {
		return models.AuthVerification{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.AuthVerification{}, err
	}
	verification.VerifiedAt = &now
	return verification, nil
}

func (r *Repository) ListAuthMFA(ctx context.Context, identityID, status string) ([]models.AuthMFAFactor, error) {
	rows, err := r.db.Query(ctx, `SELECT id,auth_identity_id,provider,status,provider_metadata,metadata,created_at,updated_at,deleted_at FROM auth_mfa_factor WHERE auth_identity_id=$1 AND deleted_at IS NULL AND ($2='' OR status=$2) ORDER BY created_at`, identityID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AuthMFAFactor{}
	for rows.Next() {
		var factor models.AuthMFAFactor
		if err = rows.Scan(&factor.ID, &factor.AuthIdentityID, &factor.Provider, &factor.Status, &factor.ProviderMetadata, &factor.Metadata, &factor.CreatedAt, &factor.UpdatedAt, &factor.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, factor)
	}
	return out, rows.Err()
}

func (r *Repository) RetrieveAuthMFA(ctx context.Context, id, identityID string) (models.AuthMFAFactor, error) {
	var factor models.AuthMFAFactor
	err := r.db.QueryRow(ctx, `SELECT id,auth_identity_id,provider,status,provider_metadata,metadata,created_at,updated_at,deleted_at FROM auth_mfa_factor WHERE id=$1 AND auth_identity_id=$2 AND deleted_at IS NULL`, id, identityID).Scan(&factor.ID, &factor.AuthIdentityID, &factor.Provider, &factor.Status, &factor.ProviderMetadata, &factor.Metadata, &factor.CreatedAt, &factor.UpdatedAt, &factor.DeletedAt)
	return factor, err
}

func (r *Repository) CreateAuthMFA(ctx context.Context, factor models.AuthMFAFactor) (models.AuthMFAFactor, error) {
	factor.ID = entityid.New("authmfa")
	err := r.db.QueryRow(ctx, `INSERT INTO auth_mfa_factor(id,auth_identity_id,provider,status,provider_metadata,metadata) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at,updated_at,deleted_at`, factor.ID, factor.AuthIdentityID, factor.Provider, factor.Status, factor.ProviderMetadata, factor.Metadata).Scan(&factor.CreatedAt, &factor.UpdatedAt, &factor.DeletedAt)
	return factor, err
}

func (r *Repository) UpdateAuthMFA(ctx context.Context, id, identityID, status string) (models.AuthMFAFactor, error) {
	var factor models.AuthMFAFactor
	err := r.db.QueryRow(ctx, `UPDATE auth_mfa_factor SET status=$3,updated_at=now() WHERE id=$1 AND auth_identity_id=$2 AND deleted_at IS NULL RETURNING id,auth_identity_id,provider,status,provider_metadata,metadata,created_at,updated_at,deleted_at`, id, identityID, status).Scan(&factor.ID, &factor.AuthIdentityID, &factor.Provider, &factor.Status, &factor.ProviderMetadata, &factor.Metadata, &factor.CreatedAt, &factor.UpdatedAt, &factor.DeletedAt)
	return factor, err
}

func (r *Repository) ListAuthMFARecoveryCodes(ctx context.Context, identityID string) ([]models.AuthMFARecoveryCode, error) {
	rows, err := r.db.Query(ctx, `SELECT id,auth_identity_id,code_hash FROM auth_mfa_recovery_code WHERE auth_identity_id=$1 AND deleted_at IS NULL ORDER BY created_at`, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	codes := []models.AuthMFARecoveryCode{}
	for rows.Next() {
		var code models.AuthMFARecoveryCode
		if err = rows.Scan(&code.ID, &code.AuthIdentityID, &code.CodeHash); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (r *Repository) ReplaceAuthMFARecoveryCodes(ctx context.Context, identityID string, hashes []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE auth_mfa_recovery_code SET deleted_at=now(),updated_at=now() WHERE auth_identity_id=$1 AND deleted_at IS NULL`, identityID); err != nil {
		return err
	}
	for _, hash := range hashes {
		if _, err = tx.Exec(ctx, `INSERT INTO auth_mfa_recovery_code(id,auth_identity_id,code_hash) VALUES($1,$2,$3)`, entityid.New("authmfarec"), identityID, hash); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteAuthMFARecoveryCode(ctx context.Context, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `UPDATE auth_mfa_recovery_code SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return tag.RowsAffected() > 0, err
}

func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func (r *Repository) FindProviderCredential(ctx context.Context, provider, email, actorType string) (domain.ProviderCredential, error) {
	var credential domain.ProviderCredential
	err := r.db.QueryRow(ctx, `SELECT p.auth_identity_id,p.provider_metadata->>'password',COALESCE(a.app_metadata->>$2,''),p.id,a.app_metadata,p.user_metadata FROM provider_identity p JOIN auth_identity a ON a.id=p.auth_identity_id WHERE p.provider=$1 AND lower(p.entity_id)=lower($3) AND a.deleted_at IS NULL`, provider, actorType+"_id", email).Scan(&credential.IdentityID, &credential.PasswordHash, &credential.ActorID, &credential.ProviderIdentityID, &credential.AppMetadata, &credential.UserMetadata)
	return credential, err
}

func (r *Repository) UpdateProviderPassword(ctx context.Context, providerIdentityID, passwordHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE provider_identity SET provider_metadata=jsonb_set(provider_metadata,'{password}',to_jsonb($2::text),true) WHERE id=$1`, providerIdentityID, passwordHash)
	return err
}

func (r *Repository) SetActorMetadata(ctx context.Context, identityID, actorType, actorID string) (string, error) {
	key := actorType + "_id"
	var current []byte
	if err := r.db.QueryRow(ctx, `SELECT app_metadata FROM auth_identity WHERE id=$1 AND deleted_at IS NULL`, identityID).Scan(&current); err != nil {
		return "", err
	}
	var metadata map[string]any
	if err := json.Unmarshal(current, &metadata); err != nil {
		return "", err
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	if _, exists := metadata[key]; exists {
		return "", fmt.Errorf("key %s already exists in app metadata", key)
	}
	metadata[key] = actorID
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	result, err := r.db.Exec(ctx, `UPDATE auth_identity SET app_metadata=$2,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, identityID, encoded)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", errors.New("auth identity not found")
	}
	return key, nil
}

func (r *Repository) UnsetActorMetadata(ctx context.Context, identityID, key string) error {
	_, err := r.db.Exec(ctx, `UPDATE auth_identity SET app_metadata=jsonb_set(COALESCE(app_metadata,'{}'::jsonb),ARRAY[$2]::text[],'null'::jsonb,true),updated_at=now() WHERE id=$1`, identityID, key)
	return err
}

var _ domain.IdentityRepository = (*Repository)(nil)
