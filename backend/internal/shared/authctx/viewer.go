// Package authctx, doğrulanmış çağıranı istek bağlamında taşır.
//
// Kendi paketinde duruyor çünkü hem GraphQL taşıma katmanı hem de resolver'lar
// onu okur; taşıma katmanında tutulsaydı resolver paketiyle içe aktarma
// döngüsü oluşurdu.
package authctx

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
)

type contextKey string

// viewerKey, doğrulanmış çağıranı taşıyan bağlam anahtarı.
const viewerKey contextKey = "prova_viewer"

// Viewer, isteği yapan doğrulanmış kullanıcı.
//
// OrgID her zaman JWT claim'inden gelir. Başlık tabanlı kiracı çözümlemesi
// bilerek yoktur: istemciden gelen bir organizasyon kimliğine güvenmek,
// kiracı sınırını istemcinin insafına bırakmak demektir.
type Viewer struct {
	UserID uuid.UUID
	Email  string
	OrgID  uuid.UUID
	// TokenID, access token'ın jti değeri. Cihaz iptalinde denylist kontrolü
	// bunun üzerinden yapılır.
	TokenID string
	// DeviceID, token'ın bağlı olduğu cihaz. Oturum başlatma bunu ister:
	// sertifika taşıyan bir sınav, bilinen bir makineye bağlanabilmeli.
	DeviceID *uuid.UUID
	// RefreshFamilyID, çıkış işleminin iptal edeceği refresh zinciri.
	RefreshFamilyID *uuid.UUID
	// ExpiresAt, access token'ın sona erme anı. Denylist girdisinin ne kadar
	// tutulacağını belirler.
	ExpiresAt time.Time
	// Roles ve Permissions token'dan gelir ama yetki kararında kullanılmaz;
	// karar her zaman RBAC servisine sorulur, çünkü token dakikalarca eski
	// olabilir ve iptal edilen bir yetki hemen etkili olmalıdır.
	Roles       []string
	Permissions []string
}

// WithViewer, çağıranı bağlama yerleştirir.
func WithViewer(ctx context.Context, v *Viewer) context.Context {
	return context.WithValue(ctx, viewerKey, v)
}

// ViewerFrom, bağlamdaki çağıranı döndürür. İkinci dönüş false ise istek
// kimliksizdir.
func ViewerFrom(ctx context.Context) (*Viewer, bool) {
	v, ok := ctx.Value(viewerKey).(*Viewer)
	return v, ok && v != nil
}

// FromClaims, token claim'lerini çağıran nesnesine çevirir.
func FromClaims(claims *service.TokenClaims) *Viewer {
	return &Viewer{
		UserID:          claims.UserID,
		Email:           claims.Email,
		OrgID:           claims.OrganizationID,
		TokenID:         claims.TokenID,
		DeviceID:        claims.DeviceID,
		RefreshFamilyID: claims.RefreshFamilyID,
		ExpiresAt:       claims.ExpiresAt,
		Roles:           claims.Roles,
		Permissions:     claims.Permissions,
	}
}

// BearerToken, Authorization başlığından token'ı ayıklar.
func BearerToken(header string) string {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// clientIPKey, isteğin kaynak adresini taşıyan bağlam anahtarı.
const clientIPKey contextKey = "prova_client_ip"

// WithClientIP, isteğin kaynak adresini bağlama yerleştirir.
//
// GraphQL resolver'ı HTTP isteğini görmez, ama hız sınırlaması ve denetim
// kaydı adrese ihtiyaç duyar. Taşıma katmanı onu buraya koyar.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

// ClientIPFrom, isteğin kaynak adresini döndürür. Bilinmiyorsa boş.
func ClientIPFrom(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey).(string)
	return ip
}
