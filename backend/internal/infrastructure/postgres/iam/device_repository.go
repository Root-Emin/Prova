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

// DeviceRepo implements repository.DeviceRepository with PostgreSQL.
type DeviceRepo struct {
	db *pgxpool.Pool
}

// NewDeviceRepo creates a new DeviceRepo.
func NewDeviceRepo(db *pgxpool.Pool) *DeviceRepo {
	return &DeviceRepo{db: db}
}

const deviceColumns = `id, user_id, fingerprint, name, platform, last_seen_at, revoked_at, created_at`

// Pair registers or refreshes the pairing and fills in the stored row.
//
// The insert-or-update runs as one statement so that two logins racing from the
// same machine cannot both be told they are the first. Which one wins does not
// matter; that exactly one of them sends the new-device warning does.
func (r *DeviceRepo) Pair(ctx context.Context, device *model.Device) (bool, error) {
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	if device.LastSeenAt.IsZero() {
		device.LastSeenAt = time.Now().UTC()
	}

	var (
		stored  model.Device
		created bool
	)
	err := r.db.QueryRow(ctx,
		`INSERT INTO user_devices (id, user_id, fingerprint, name, platform, last_seen_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)
		 ON CONFLICT (user_id, fingerprint) DO UPDATE
		   SET last_seen_at = EXCLUDED.last_seen_at,
		       -- Keep the stored name unless the client supplied a new one, so
		       -- a client that forgets to send it cannot blank the list entry.
		       name = COALESCE(NULLIF(EXCLUDED.name, ''), user_devices.name)
		 RETURNING `+deviceColumns+`, (xmax = 0) AS created`,
		device.ID, device.UserID, device.Fingerprint, device.Name, device.Platform, device.LastSeenAt,
	).Scan(&stored.ID, &stored.UserID, &stored.Fingerprint, &stored.Name, &stored.Platform,
		&stored.LastSeenAt, &stored.RevokedAt, &stored.CreatedAt, &created)
	if err != nil {
		return false, domainErr.New(domainErr.ErrInternal, "failed to pair device", err)
	}

	*device = stored
	return created, nil
}

func (r *DeviceRepo) GetByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*model.Device, error) {
	var d model.Device
	err := r.db.QueryRow(ctx,
		`SELECT `+deviceColumns+` FROM user_devices WHERE user_id = $1 AND fingerprint = $2`,
		userID, fingerprint,
	).Scan(&d.ID, &d.UserID, &d.Fingerprint, &d.Name, &d.Platform, &d.LastSeenAt, &d.RevokedAt, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "device not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get device", err)
	}
	return &d, nil
}

func (r *DeviceRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.Device, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+deviceColumns+` FROM user_devices WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list devices", err)
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		var d model.Device
		if err := rows.Scan(&d.ID, &d.UserID, &d.Fingerprint, &d.Name, &d.Platform,
			&d.LastSeenAt, &d.RevokedAt, &d.CreatedAt); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan device", err)
		}
		devices = append(devices, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list devices", err)
	}
	return devices, nil
}

// Revoke matches on user_id as well as id, so a guessed device identifier from
// another account revokes nothing.
func (r *DeviceRepo) Revoke(ctx context.Context, userID, deviceID uuid.UUID, at time.Time) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE user_devices SET revoked_at = $1 WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL`,
		at, deviceID, userID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to revoke device", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "device not found", nil)
	}
	return nil
}

func (r *DeviceRepo) TouchLastSeen(ctx context.Context, deviceID uuid.UUID, at time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE user_devices SET last_seen_at = $1 WHERE id = $2`, at, deviceID)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to update device", err)
	}
	return nil
}
