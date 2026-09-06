package model

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
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
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
	// Fingerprint, istemcinin gönderdiği ham parmak izi. Yalnızca bellekte
	// yaşar; depoya FingerprintHash yazılır.
	Fingerprint string `json:"-"`
	// FingerprintHash, parmak izinin saklanan biçimi. Parmak izi bir sırdır:
	// düz metin saklanırsa veritabanına erişen biri onu istemci gibi
	// gönderebilir.
	FingerprintHash string `json:"-"`
	// PublicKey, cihazın ürettiği anahtar çiftinin genel yarısı (base64,
	// Ed25519). Özel anahtar işletim sisteminin güvenli deposunda kalır ve
	// sunucuya hiç gelmez.
	PublicKey string         `json:"-"`
	Name      string         `json:"name"`
	Platform  DevicePlatform `json:"platform"`
	// IPAddress is taken from the authenticated HTTP request, not from the
	// Desktop payload. It is the most recently observed source address.
	IPAddress string `json:"ip_address"`
	// MACAddress is collected by Desktop from its active network interface and
	// is retained for the organisation's device-integrity record.
	MACAddress string     `json:"mac_address"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// HasKeyPair reports whether the device can answer a signature challenge.
func (d *Device) HasKeyPair() bool { return d.PublicKey != "" }

// IsRevoked reports whether the user has withdrawn this device's authorisation.
func (d *Device) IsRevoked() bool {
	return d.RevokedAt != nil
}

// NormalizePlatform maps a client-supplied platform string onto a known value.
//
// Electron'un doğal platform değerleri Node'un process.platform'undan gelir:
// "win32", "darwin", "linux". Eski sürüm yalnızca kanonik adları tanıyordu, bu
// yüzden gerçek masaüstü istemcisinden gelen her cihaz "unknown" olarak
// kaydediliyordu — ve cihaz listesi kullanıcı için okunamaz hâle geliyordu.
// Tarayıcının navigator.platform değerleri de ("Win32", "MacIntel", "Linux
// x86_64") burada karşılanıyor.
func NormalizePlatform(s string) DevicePlatform {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch {
	case normalized == "":
		return DevicePlatformUnknown
	case strings.HasPrefix(normalized, "win"):
		return DevicePlatformWindows
	case strings.HasPrefix(normalized, "darwin"),
		strings.HasPrefix(normalized, "mac"),
		strings.HasPrefix(normalized, "osx"),
		strings.HasPrefix(normalized, "os x"):
		return DevicePlatformMacOS
	case strings.HasPrefix(normalized, "linux"),
		strings.HasPrefix(normalized, "freebsd"),
		strings.HasPrefix(normalized, "openbsd"),
		strings.HasPrefix(normalized, "x11"):
		return DevicePlatformLinux
	case strings.HasPrefix(normalized, "web"), strings.HasPrefix(normalized, "browser"):
		return DevicePlatformWeb
	default:
		return DevicePlatformUnknown
	}
}

// NormalizeIPAddress accepts only a literal IPv4 or IPv6 address. The server
// gets it from request middleware, so malformed forwarded-header values never
// enter the device inventory.
func NormalizeIPAddress(value string) string {
	parsed := net.ParseIP(strings.TrimSpace(value))
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

// NormalizeMACAddress canonicalises a real, six-byte unicast adapter address
// for display and storage. A MAC is supplied by the Desktop client, therefore
// invalid, multicast, broadcast, and all-zero values are ignored.
func NormalizeMACAddress(value string) string {
	parsed, err := net.ParseMAC(strings.TrimSpace(value))
	if err != nil || len(parsed) != 6 || parsed[0]&1 == 1 {
		return ""
	}
	allZero := true
	for _, octet := range parsed {
		if octet != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ""
	}
	return strings.ToUpper(parsed.String())
}

// HashFingerprint, parmak izinin saklanacak biçimini üretir.
//
// Pepper yok ve gerekmiyor: giriş kodunun aksine parmak izi yüksek entropilidir
// ve tahmin edilecek küçük bir uzayı yoktur. Hash'in tek işi, veritabanına
// erişen birinin parmak izini istemci gibi geri gönderememesini sağlamak.
func HashFingerprint(fingerprint string) string {
	if fingerprint == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(sum[:])
}
