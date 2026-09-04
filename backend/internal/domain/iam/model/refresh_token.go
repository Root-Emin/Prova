package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken, rotasyonlu yenileme zincirinin bir halkası.
//
// FamilyID, aynı giriş oturumundan türeyen tüm halkaları bağlar. Kullanılmış
// bir token tekrar geldiğinde ailenin tamamı iptal edilir: ya token çalınmış
// ve saldırgan kullanıyor, ya da meşru kullanıcı eski bir kopyayı yeniden
// oynatıyor. İkisini ayırt etmenin bir yolu yok, ve güvenli taraf ikisini de
// kesmek.
type RefreshToken struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	FamilyID uuid.UUID `json:"family_id"`
	// TokenHash, token'ın SHA-256 özeti. Token'ın kendisi saklanmaz.
	TokenHash      string     `json:"-"`
	DeviceID       *uuid.UUID `json:"device_id,omitempty"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	// ParentID, bu halkanın türediği token. Zincirin okunabilir bir geçmişi
	// olsun diye tutuluyor: bir aile iptal edildiğinde hangi halkanın
	// yeniden oynatıldığını göstermek gerekir.
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsUsable reports whether the token may still be exchanged.
func (t *RefreshToken) IsUsable(now time.Time) bool {
	return t.UsedAt == nil && t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

// IsReplay reports whether this token has already been spent.
//
// Yeniden kullanım, süresi dolmuş ya da iptal edilmiş bir token'dan farklı
// bir olay: ilki bir saldırı belirtisi, diğerleri olağan yaşam döngüsü.
// Ayırt edilmezse aileyi iptal etme kararı da verilemez.
func (t *RefreshToken) IsReplay(now time.Time) bool {
	return t.UsedAt != nil && t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
