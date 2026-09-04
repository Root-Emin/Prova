package graph

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/graph/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	"github.com/masterfabric-go/masterfabric/internal/shared/authctx"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// viewer, bağlamdaki doğrulanmış çağıranı döndürür.
//
// @auth directive'i zaten kapıyı tutuyor; buradaki kontrol ikinci savunma
// hattıdır. Şemada @auth yazmayı unutmuş bir alan, resolver'da sessizce
// kimliksiz çalışmak yerine hata vermeli.
func viewer(ctx context.Context) (*authctx.Viewer, error) {
	v, ok := authctx.ViewerFrom(ctx)
	if !ok {
		return nil, domainErr.New(domainErr.ErrUnauthorized, "oturum açmanız gerekiyor", nil)
	}
	return v, nil
}

// notImplemented, henüz karşılığı yazılmamış alanlar için.
//
// panic yerine hata döndürülüyor: üretilen resolver iskeleti panic ediyor ve
// tek bir eksik alan tüm süreci düşürebilir. Recover handler'ı bunu yakalasa
// bile, hata olarak dönmek istemciye ne olduğunu söyler.
func notImplemented(field string) error {
	return domainErr.New(domainErr.ErrNotImplemented, field+" bu sürümde kullanılamıyor", nil)
}

// intOrDefault, isteğe bağlı sayfalama argümanlarını çözer.
func intOrDefault(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

// boolOrDefault, isteğe bağlı bayrakları çözer.
func boolOrDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

// clampLimit, istemcinin istediği sayfa boyutunu makul bir tavana çeker.
//
// Limit istemciden geldiği için sınırsız bırakılamaz: tek bir sorgu tüm
// koleksiyonu belleğe çekebilir.
func clampLimit(requested, fallback, max int) int {
	if requested <= 0 {
		requested = fallback
	}
	if requested > max {
		return max
	}
	return requested
}

// permissionsOf, RBAC servisinden çağıranın izinlerini okur.
func (r *Resolver) permissionsOf(ctx context.Context, userID, orgID uuid.UUID) []string {
	if r.RBAC == nil {
		return nil
	}
	permissions, err := r.RBAC.GetUserPermissions(ctx, userID, orgID)
	if err != nil {
		// İzin listesi görüntüleme amaçlıdır; okunamaması sorguyu
		// patlatmamalı. Yetki kararları zaten @permission directive'inde
		// ayrıca veriliyor.
		r.Log.ErrorContext(ctx, "izinler okunamadı", "user_id", userID, "error", err)
		return nil
	}
	return permissions
}

// derefString, isteğe bağlı metin argümanlarını çözer.
func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// clientIP, hız sınırlaması için isteğin kaynağını döndürür.
//
// GraphQL resolver'ı HTTP isteğini doğrudan görmez; adres taşıma katmanında
// bağlama konur. Yoksa boş döner ve IP'ye bağlı limitler devre dışı kalır —
// adrese bağlı limitler yine çalışır.
func clientIP(ctx context.Context) string {
	return authctx.ClientIPFrom(ctx)
}

var _ = model.UserStatusActive

// scopeOf, çağıranın kiracı sınırını üretir.
//
// Her Prova deposu bunu istiyor ve org kimliği yalnızca JWT claim'inden
// geliyor: istemciden gelen bir organizasyon kimliğine güvenmek, kiracı
// sınırını istemcinin insafına bırakmak olurdu.
func scopeOf(v *authctx.Viewer) provaRepo.Scope {
	return provaRepo.NewScope(v.OrgID)
}

// hasPermission, RBAC servisine tek bir izni sorar.
//
// @permission directive'i alan bazında çalışıyor; bu yardımcı, izne alan
// seviyesinde değil kayıt seviyesinde bakılması gereken yerler için
// (başkasının oturumunu okumak gibi).
func (r *Resolver) hasPermission(ctx context.Context, v *authctx.Viewer, permission string) (bool, error) {
	if r.RBAC == nil {
		return false, nil
	}
	return r.RBAC.HasPermission(ctx, v.UserID, v.OrgID, permission)
}
