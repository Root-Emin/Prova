package model

import (
	"time"

	"github.com/google/uuid"
)

// DeviceChallenge, cihazın imzalaması istenen tek kullanımlık meydan okuma.
//
// Parmak izi kopyalanabilir bir dizedir: veritabanına ya da istemci koduna
// erişen biri onu olduğu gibi gönderebilir. İmza kopyalanamaz, çünkü özel
// anahtar işletim sisteminin güvenli deposundan hiç çıkmaz. Challenge her
// girişte yeniden üretilir ve tek kullanımlıktır; sabit bir metni imzalatmak,
// imzayı yeniden oynatılabilir bir parolaya çevirirdi.
type DeviceChallenge struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"-"`
	// FingerprintHash, challenge'ın bağlandığı cihaz. Bir cihaz için
	// üretilen challenge başka bir cihazla kullanılamaz.
	FingerprintHash string `json:"-"`
	// Challenge, imzalanacak rastgele metin (base64).
	Challenge  string     `json:"challenge"`
	ExpiresAt  time.Time  `json:"expires_at"`
	ConsumedAt *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
}

// IsUsable reports whether the challenge may still be answered.
func (c *DeviceChallenge) IsUsable(now time.Time) bool {
	return c.ConsumedAt == nil && now.Before(c.ExpiresAt)
}
