package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Token URL'e doğrudan girecek; yüzde-kodlama gerektiren bir token, kırılan
// bir bağlantı ve çalışmayan bir giriş demektir.
func TestMagicLinkService_GeneratesURLSafeTokens(t *testing.T) {
	svc := NewMagicLinkService()

	token, _, err := svc.Generate()
	if err != nil {
		t.Fatalf("üretim başarısız: %v", err)
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("token URL-güvenli olmalı: %q", token)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token çözülemedi: %v", err)
	}
	if len(decoded) != magicLinkTokenBytes {
		t.Fatalf("%d bayt entropi beklenir, %d geldi", magicLinkTokenBytes, len(decoded))
	}
}

// Depoya token değil hash'i yazılıyor: veritabanı sızarsa sızan şey kimlik
// bilgisi olmamalı.
func TestMagicLinkService_StoredHashIsNotTheToken(t *testing.T) {
	svc := NewMagicLinkService()

	token, hash, err := svc.Generate()
	if err != nil {
		t.Fatalf("üretim başarısız: %v", err)
	}
	if hash == token || strings.Contains(hash, token) {
		t.Fatal("saklanan değer token'ın kendisi olmamalı")
	}
	if svc.Hash(token) != hash {
		t.Fatal("hash deterministik olmalı, yoksa doğrulama hiç eşleşmez")
	}
}

func TestMagicLinkService_TokensAreUnique(t *testing.T) {
	svc := NewMagicLinkService()

	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		token, _, err := svc.Generate()
		if err != nil {
			t.Fatalf("üretim başarısız: %v", err)
		}
		if seen[token] {
			t.Fatal("token'lar benzersiz olmalı")
		}
		seen[token] = true
	}
}
