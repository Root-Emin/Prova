package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// DeviceRepository persists the machines paired with user accounts.
type DeviceRepository interface {
	// Pair registers the fingerprint against the user, or refreshes an existing
	// pairing. It reports created=true only the first time a fingerprint is
	// seen for that user, which is what triggers the new-device notification.
	Pair(ctx context.Context, device *model.Device) (created bool, err error)

	// GetByFingerprint returns the pairing for a user's fingerprint, or
	// ErrNotFound.
	GetByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*model.Device, error)

	// ListByUser returns every device paired with the user, revoked ones
	// included, newest first.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.Device, error)

	// Revoke withdraws a device's authorisation. Ownership is enforced through
	// userID so that one user cannot revoke another's device.
	Revoke(ctx context.Context, userID, deviceID uuid.UUID, at time.Time) error

	// TouchLastSeen records that the device was used.
	TouchLastSeen(ctx context.Context, deviceID uuid.UUID, at time.Time) error
}
