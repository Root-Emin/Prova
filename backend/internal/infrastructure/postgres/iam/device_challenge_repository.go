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

// DeviceChallengeRepo implements repository.DeviceChallengeRepository.
type DeviceChallengeRepo struct {
	db *pgxpool.Pool
}

// NewDeviceChallengeRepo creates a new DeviceChallengeRepo.
func NewDeviceChallengeRepo(db *pgxpool.Pool) *DeviceChallengeRepo {
	return &DeviceChallengeRepo{db: db}
}

func (r *DeviceChallengeRepo) Create(ctx context.Context, challenge *model.DeviceChallenge) error {
	if challenge.ID == uuid.Nil {
		challenge.ID = uuid.New()
	}
	if challenge.CreatedAt.IsZero() {
		challenge.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO device_challenges (id, email, fingerprint_hash, challenge, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		challenge.ID, challenge.Email, challenge.FingerprintHash,
		challenge.Challenge, challenge.ExpiresAt, challenge.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "cihaz challenge'ı oluşturulamadı", err)
	}
	return nil
}

func (r *DeviceChallengeRepo) GetLatestUsable(ctx context.Context, email, fingerprintHash string, now time.Time) (*model.DeviceChallenge, error) {
	var c model.DeviceChallenge
	err := r.db.QueryRow(ctx,
		`SELECT id, email, fingerprint_hash, challenge, expires_at, consumed_at, created_at
		   FROM device_challenges
		  WHERE lower(email) = lower($1) AND fingerprint_hash = $2
		    AND consumed_at IS NULL AND expires_at > $3
		  ORDER BY created_at DESC
		  LIMIT 1`,
		email, fingerprintHash, now,
	).Scan(&c.ID, &c.Email, &c.FingerprintHash, &c.Challenge, &c.ExpiresAt, &c.ConsumedAt, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "kullanılabilir challenge yok", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "challenge okunamadı", err)
	}
	return &c, nil
}

func (r *DeviceChallengeRepo) Consume(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE device_challenges SET consumed_at = $1
		  WHERE id = $2 AND consumed_at IS NULL AND expires_at > $1`,
		at, id,
	)
	if err != nil {
		return false, domainErr.New(domainErr.ErrInternal, "challenge yakılamadı", err)
	}
	return tag.RowsAffected() == 1, nil
}
