package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	"github.com/redis/go-redis/v9"
)

// Denylist, imzası hâlâ geçerli olan token'ları reddeder.
//
// JWT'nin doğası gereği bir token, süresi dolana kadar geçerlidir; iptal
// ancak dışarıdan tutulan bir listeyle mümkündür. Liste iki eksende çalışır:
// tek token (çıkış) ve tüm cihaz (cihaz iptali). İkincisi olmadan çalınmış bir
// makinenin oturumu, cihaz kayıtta iptal edilmiş olsa bile access token'ın
// ömrü boyunca yaşamaya devam ederdi.
type Denylist struct {
	redis *redis.Client
	// memory, Redis yokken kullanılan yedek. Tek örneklik geliştirme
	// kurulumunda doğru çalışır; çok örnekli dağıtımda iptal yalnızca
	// isteği alan örnekte geçerli olur, bu yüzden üretimde Redis şart.
	memory *memoryDenylist
}

// NewDenylist wires the denylist over Redis, falling back to memory.
func NewDenylist(client *redis.Client) *Denylist {
	d := &Denylist{redis: client}
	if client == nil {
		d.memory = newMemoryDenylist()
	}
	return d
}

func tokenKey(tokenID string) string { return "denylist:jti:" + tokenID }
func deviceKey(id uuid.UUID) string  { return "denylist:device:" + id.String() }
func familyKey(id uuid.UUID) string  { return "denylist:family:" + id.String() }

// RevokeToken, tek bir access token'ı listeye alır. Çıkış işleminin karşılığı.
func (d *Denylist) RevokeToken(ctx context.Context, tokenID string, until time.Time) error {
	if tokenID == "" {
		return nil
	}
	return d.set(ctx, tokenKey(tokenID), until)
}

// RevokeDeviceTokens, bir cihaza bağlı her token'ı reddeder.
func (d *Denylist) RevokeDeviceTokens(ctx context.Context, _ uuid.UUID, deviceID uuid.UUID, until time.Time) error {
	return d.set(ctx, deviceKey(deviceID), until)
}

// RevokeFamily, bir refresh token ailesini reddeder.
func (d *Denylist) RevokeFamily(ctx context.Context, familyID uuid.UUID, until time.Time) error {
	return d.set(ctx, familyKey(familyID), until)
}

// IsFamilyRevoked reports whether the refresh family has been killed.
func (d *Denylist) IsFamilyRevoked(ctx context.Context, familyID uuid.UUID) (bool, error) {
	return d.exists(ctx, familyKey(familyID))
}

// IsRevoked, token'ın kendisi ya da bağlı olduğu cihaz iptal edilmiş mi.
func (d *Denylist) IsRevoked(ctx context.Context, claims *service.TokenClaims) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.TokenID != "" {
		revoked, err := d.exists(ctx, tokenKey(claims.TokenID))
		if err != nil || revoked {
			return revoked, err
		}
	}
	if claims.DeviceID != nil {
		return d.exists(ctx, deviceKey(*claims.DeviceID))
	}
	return false, nil
}

func (d *Denylist) set(ctx context.Context, key string, until time.Time) error {
	ttl := time.Until(until)
	if ttl <= 0 {
		// Zaten sona ermiş bir token'ı listeye almanın etkisi yok.
		return nil
	}
	if d.memory != nil {
		d.memory.set(key, until)
		return nil
	}
	if err := d.redis.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("iptal listesine yazılamadı: %w", err)
	}
	return nil
}

func (d *Denylist) exists(ctx context.Context, key string) (bool, error) {
	if d.memory != nil {
		return d.memory.exists(key), nil
	}
	err := d.redis.Get(ctx, key).Err()
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, redis.Nil):
		return false, nil
	default:
		return false, fmt.Errorf("iptal listesi okunamadı: %w", err)
	}
}

// memoryDenylist, Redis yokken kullanılan süreç içi yedek.
type memoryDenylist struct {
	mu      sync.RWMutex
	entries map[string]time.Time
}

func newMemoryDenylist() *memoryDenylist {
	return &memoryDenylist{entries: map[string]time.Time{}}
}

func (m *memoryDenylist) set(key string, until time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[key] = until
	// Süresi dolmuş girdiler burada temizleniyor: ayrı bir zamanlayıcı
	// çalıştırmak, en fazla birkaç yüz girdi tutan bir yapı için gereksiz.
	now := time.Now()
	for k, expiry := range m.entries {
		if now.After(expiry) {
			delete(m.entries, k)
		}
	}
}

func (m *memoryDenylist) exists(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	expiry, ok := m.entries[key]
	return ok && time.Now().Before(expiry)
}
