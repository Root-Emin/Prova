package model

import (
	"time"

	"github.com/google/uuid"
)

// MagicLink, e-postadaki tek kullanımlık giriş bağlantısı.
//
// Kod hattıyla aynı ilkeyi izler: token'ın kendisi saklanmaz, yalnızca
// hash'i. Koddan farkı pepper'a ihtiyaç duymaması — kod uzayı 10^6 iken
// token 256 bit kriptografik rastgelelik taşır, ve o uzayda gökkuşağı
// tablosu diye bir şey yoktur.
type MagicLink struct {
	ID     uuid.UUID  `json:"id"`
	UserID *uuid.UUID `json:"user_id,omitempty"`
	Email  string     `json:"email"`
	// TokenHash, e-postadaki token'ın SHA-256 özeti.
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	RequestIP string     `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsRedeemable reports whether the link may still be exchanged for a session.
func (m *MagicLink) IsRedeemable(now time.Time) bool {
	return m.UsedAt == nil && m.RevokedAt == nil && now.Before(m.ExpiresAt)
}
