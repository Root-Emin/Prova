package dto

import (
	"time"

	"github.com/google/uuid"
)

// RequestLoginCodeRequest asks for a one-time code to be mailed to an address.
type RequestLoginCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// RequestLoginCodeResponse is what the caller gets back.
//
// It carries no signal about whether the address belongs to an account. The
// same body is returned for a registered address, an unknown one, and one that
// is currently rate limited — otherwise the endpoint becomes a membership
// oracle for anyone with a list of corporate addresses.
type RequestLoginCodeResponse struct {
	// Sent is always true. It means "the request was accepted", not "an e-mail
	// left the building".
	Sent bool `json:"sent"`
	// ExpiresInSeconds lets the client show a countdown. It is a constant.
	ExpiresInSeconds int `json:"expires_in_seconds"`
	// ResendAfterSeconds is how long the client should disable its resend
	// button. Also a constant.
	ResendAfterSeconds int `json:"resend_after_seconds"`
}

// DeviceInfo is the desktop client's description of the machine it runs on.
//
// Optional: the web panel sends nothing. The desktop app sends a fingerprint,
// and that is what pairs the machine with the account at login.
type DeviceInfo struct {
	Fingerprint string `json:"fingerprint" validate:"omitempty,min=16,max=128"`
	Name        string `json:"name" validate:"omitempty,max=255"`
	Platform    string `json:"platform" validate:"omitempty,max=32"`
	// PublicKey, cihazın ürettiği anahtar çiftinin genel yarısı (base64,
	// Ed25519). Yalnızca ilk kayıtta gönderilir; özel anahtar işletim
	// sisteminin güvenli deposunda kalır ve sunucuya hiç gelmez.
	PublicKey string `json:"public_key" validate:"omitempty,max=256"`
}

// VerifyLoginCodeRequest redeems a code for a session.
type VerifyLoginCodeRequest struct {
	Email  string      `json:"email" validate:"required,email"`
	Code   string      `json:"code" validate:"required,min=4,max=12"`
	Device *DeviceInfo `json:"device,omitempty"`
	// DeviceSignature, sunucunun verdiği challenge'ın cihaz özel anahtarıyla
	// imzası (base64). Kayıtlı anahtarı olan cihazlarda zorunludur: parmak
	// izi kopyalanabilir, imza kopyalanamaz.
	DeviceSignature string `json:"device_signature" validate:"omitempty,max=256"`
}

// VerifyLoginCodeResponse is the successful sign-in result.
type VerifyLoginCodeResponse struct {
	Token string `json:"token"`
	// RefreshToken, rotasyonlu yenileme zincirinin ilk halkası. Access token
	// kısa ömürlüdür; oturumun devamı bu değere bağlıdır.
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	// OrganizationID, token'a yazılan etkin kiracı. İstemci bunu görüntüler;
	// yetki kararları yine sunucuda claim üzerinden verilir.
	OrganizationID uuid.UUID     `json:"organization_id"`
	User           UserInfo      `json:"user"`
	Device         *PairedDevice `json:"device,omitempty"`
}

// VerifyMagicLinkRequest redeems a one-time link for a session.
type VerifyMagicLinkRequest struct {
	Token           string      `json:"token" validate:"required"`
	Device          *DeviceInfo `json:"device,omitempty"`
	DeviceSignature string      `json:"device_signature" validate:"omitempty,max=256"`
}

// DeviceChallengeRequest asks for a signature challenge.
type DeviceChallengeRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Fingerprint string `json:"fingerprint" validate:"required,min=16,max=128"`
}

// DeviceChallengeResponse carries the text the device must sign.
type DeviceChallengeResponse struct {
	Challenge string    `json:"challenge"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PairedDevice describes the device bound to the account during this sign-in.
type PairedDevice struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Platform string    `json:"platform"`
	// IsNew marks the first sign-in from this machine, which is also what
	// triggered the notification e-mail.
	IsNew bool `json:"is_new"`
}
