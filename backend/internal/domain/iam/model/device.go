package model

import (
	"time"

	"github.com/google/uuid"
)

// DevicePlatform identifies the operating system a device runs.
type DevicePlatform string

const (
	DevicePlatformWindows DevicePlatform = "windows"
	DevicePlatformMacOS   DevicePlatform = "macos"
	DevicePlatformLinux   DevicePlatform = "linux"
	DevicePlatformWeb     DevicePlatform = "web"
	DevicePlatformUnknown DevicePlatform = "unknown"
)

// Device is a machine paired with a user account.
//
// Pairing happens the moment an e-mail login completes. The fingerprint is
// never an identity on its own — it is a second constraint layered on top of a
// verified mailbox. What it buys is exam integrity: a certificate is only
// meaningful if the session behind it can be tied to a known machine.
type Device struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Fingerprint string         `json:"-"`
	Name        string         `json:"name"`
	Platform    DevicePlatform `json:"platform"`
	LastSeenAt  time.Time      `json:"last_seen_at"`
	RevokedAt   *time.Time     `json:"revoked_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// IsRevoked reports whether the user has withdrawn this device's authorisation.
func (d *Device) IsRevoked() bool {
	return d.RevokedAt != nil
}

// NormalizePlatform maps a client-supplied platform string onto a known value.
func NormalizePlatform(s string) DevicePlatform {
	switch DevicePlatform(s) {
	case DevicePlatformWindows, DevicePlatformMacOS, DevicePlatformLinux, DevicePlatformWeb:
		return DevicePlatform(s)
	default:
		return DevicePlatformUnknown
	}
}
