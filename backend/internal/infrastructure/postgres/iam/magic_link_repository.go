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

// MagicLinkRepo implements repository.MagicLinkRepository with PostgreSQL.
type MagicLinkRepo struct {
	db *pgxpool.Pool
}

// NewMagicLinkRepo creates a new MagicLinkRepo.
func NewMagicLinkRepo(db *pgxpool.Pool) *MagicLinkRepo {
	return &MagicLinkRepo{db: db}
}

const magicLinkColumns = `id, user_id, email, token_hash, expires_at, used_at, revoked_at, COALESCE(request_ip, ''), created_at`

func (r *MagicLinkRepo) Create(ctx context.Context, link *model.MagicLink) error {
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO magic_links (id, user_id, email, token_hash, expires_at, request_ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)`,
		link.ID, link.UserID, link.Email, link.TokenHash, link.ExpiresAt, link.RequestIP, link.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "magic link oluşturulamadı", err)
	}
	return nil
}

func (r *MagicLinkRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.MagicLink, error) {
	var link model.MagicLink
	err := r.db.QueryRow(ctx,
		`SELECT `+magicLinkColumns+` FROM magic_links WHERE token_hash = $1`, tokenHash,
	).Scan(&link.ID, &link.UserID, &link.Email, &link.TokenHash, &link.ExpiresAt,
		&link.UsedAt, &link.RevokedAt, &link.RequestIP, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "magic link bulunamadı", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "magic link okunamadı", err)
	}
	return &link, nil
}

// Consume, bağlantıyı tek kullanımlık olarak yakar.
//
// Koşul sorgunun içinde: "önce oku sonra işaretle" biçiminde yazılsaydı,
// aynı bağlantıya gelen iki eşzamanlı istek iki oturum üretebilirdi.
func (r *MagicLinkRepo) Consume(ctx context.Context, id string, at time.Time) (bool, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return false, domainErr.New(domainErr.ErrValidation, "geçersiz magic link kimliği", err)
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE magic_links SET used_at = $1
		  WHERE id = $2 AND used_at IS NULL AND revoked_at IS NULL AND expires_at > $1`,
		at, parsed,
	)
	if err != nil {
		return false, domainErr.New(domainErr.ErrInternal, "magic link yakılamadı", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *MagicLinkRepo) InvalidateActive(ctx context.Context, email string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE magic_links SET revoked_at = $1
		  WHERE lower(email) = lower($2) AND used_at IS NULL AND revoked_at IS NULL`,
		at, email,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "eski magic linkler iptal edilemedi", err)
	}
	return nil
}

func (r *MagicLinkRepo) DeleteByUser(ctx context.Context, userID string) error {
	parsed, err := uuid.Parse(userID)
	if err != nil {
		return domainErr.New(domainErr.ErrValidation, "geçersiz kullanıcı kimliği", err)
	}
	if _, err := r.db.Exec(ctx, `DELETE FROM magic_links WHERE user_id = $1`, parsed); err != nil {
		return domainErr.New(domainErr.ErrInternal, "magic linkler silinemedi", err)
	}
	return nil
}
