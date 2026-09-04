package service

import (
	"context"

	"github.com/google/uuid"
)

// MembershipResolver, bir kullanıcının token'ına yazılacak etkin
// organizasyonu çözer.
//
// Bu port olmadan giriş akışı JWT'ye sıfır UUID yazıyordu; RBAC de o sıfır
// organizasyona sorgu attığı için hiçbir kullanıcı hiçbir izne sahip
// olamıyordu ve tüm korumalı uçlar 403 dönüyordu. Kimlik doğrulamanın son
// adımı, "bu kişi kim" kadar "bu kişi hangi kiracıda" sorusunu da yanıtlamak
// zorundadır.
type MembershipResolver interface {
	// ResolveActiveOrg, kullanıcının etkin organizasyonunu döndürür. Hiç
	// üyelik yoksa kişisel bir organizasyon oluşturup bağlar, böylece giriş
	// akışı asla organizasyonsuz bir token üretmez.
	ResolveActiveOrg(ctx context.Context, userID uuid.UUID, email string) (uuid.UUID, error)
}
