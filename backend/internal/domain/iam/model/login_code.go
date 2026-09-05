package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// LoginCodePurpose distinguishes the flows a code may be issued for. Codes are
// bound to a purpose so that a code mailed for one action cannot be replayed
// into another.
type LoginCodePurpose string

const (
	// LoginCodePurposeLogin authenticates an already registered, verified
	// account. It never changes e-mail verification state.
	LoginCodePurposeLogin LoginCodePurpose = "login"
)

// LoginCode is a one-time numeric code issued to an e-mail address.
//
// The public request is keyed by address to preserve an enumeration-safe API;
// hardened records also carry the immutable account identity.
//
// The code itself is never stored. Only CodeDigest is, and it is a peppered
// digest: the code space is 10^6, so an unpeppered digest column is a rainbow
// table away from plaintext.
type LoginCode struct {
	ID uuid.UUID `json:"id"`
	// UserID is the immutable account identity used by the hardened digest and
	// by the Redis key. It is populated before a challenge is issued.
	UserID      uuid.UUID        `json:"user_id"`
	Email       string           `json:"email"`
	CodeDigest  string           `json:"-"`
	Purpose     LoginCodePurpose `json:"purpose"`
	Attempts    int              `json:"attempts"`
	MaxAttempts int              `json:"max_attempts"`
	ExpiresAt   time.Time        `json:"expires_at"`
	ConsumedAt  *time.Time       `json:"consumed_at,omitempty"`
	RequestIP   string           `json:"-"`
	CreatedAt   time.Time        `json:"created_at"`
}

// NormalizeEmail folds an address to the form used as a lookup key.
//
// Addresses are compared case-insensitively and without surrounding
// whitespace. Without this, "User@corp.com" and "user@corp.com" would be two
// accounts, and rate limits keyed on the address would be trivially bypassed by
// varying the case.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// IsExpired reports whether the code is past its lifetime at the given moment.
func (c *LoginCode) IsExpired(now time.Time) bool {
	return !now.Before(c.ExpiresAt)
}

// IsConsumed reports whether the code has already been redeemed.
func (c *LoginCode) IsConsumed() bool {
	return c.ConsumedAt != nil
}

// IsExhausted reports whether the attempt budget is used up.
func (c *LoginCode) IsExhausted() bool {
	return c.MaxAttempts > 0 && c.Attempts >= c.MaxAttempts
}

// IsRedeemable reports whether the code may still be checked against user
// input.
func (c *LoginCode) IsRedeemable(now time.Time) bool {
	return !c.IsConsumed() && !c.IsExpired(now) && !c.IsExhausted()
}
