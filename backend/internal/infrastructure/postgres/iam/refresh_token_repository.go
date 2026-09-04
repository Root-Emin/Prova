package iam

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// RefreshTokenRepo implements repository.RefreshTokenRepository.
type RefreshTokenRepo struct {
	db *pgxpool.Pool
}

// NewRefreshTokenRepo creates a new RefreshTokenRepo.
func NewRefreshTokenRepo(db *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

const refreshColumns = `id, user_id, family_id, token_hash, device_id, organization_id, parent_id, expires_at, used_at, revoked_at, created_at`

func (r *RefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.FamilyID == uuid.Nil {
		token.FamilyID = token.ID
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens
		   (id, user_id, family_id, token_hash, device_id, organization_id, parent_id, expires_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		token.ID, token.UserID, token.FamilyID, token.TokenHash, token.DeviceID,
		token.OrganizationID, token.ParentID, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "refresh token yazılamadı", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	err := r.db.QueryRow(ctx,
		`SELECT `+refreshColumns+` FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.FamilyID, &t.TokenHash, &t.DeviceID, &t.OrganizationID,
		&t.ParentID, &t.ExpiresAt, &t.UsedAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "refresh token bulunamadı", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "refresh token okunamadı", err)
	}
	return &t, nil
}

// MarkUsed, halkayı harcanmış işaretler.
func (r *RefreshTokenRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE refresh_tokens SET used_at = $1
		  WHERE id = $2 AND used_at IS NULL AND revoked_at IS NULL AND expires_at > $1`,
		at, id,
	)
	if err != nil {
		return false, domainErr.New(domainErr.ErrInternal, "refresh token harcanamadı", err)
	}
	return tag.RowsAffected() == 1, nil
}

// RevokeFamily, ailenin tüm halkalarını iptal eder.
func (r *RefreshTokenRepo) RevokeFamily(ctx context.Context, familyID uuid.UUID, at time.Time) (int, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`,
		at, familyID,
	)
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "refresh token ailesi iptal edilemedi", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *RefreshTokenRepo) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
		return domainErr.New(domainErr.ErrInternal, "refresh token'lar silinemedi", err)
	}
	return nil
}
