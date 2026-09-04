// Package graph, GraphQL yüzeyini uygulama katmanına bağlar.
//
// Resolver'lar iş kuralı içermez. Görevleri üç adımdır: bağlamdaki çağıranı
// oku, ilgili use case'i çağır, sonucu şema modeline çevir. Kural resolver'a
// sızdığı anda aynı kural REST, zamanlanmış iş ve test yollarında farklı
// davranmaya başlar.
package graph

import (
	"log/slog"

	iamUC "github.com/masterfabric-go/masterfabric/internal/application/iam/usecase"
	auditRepo "github.com/masterfabric-go/masterfabric/internal/domain/audit/repository"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	iamService "github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
)

// Resolver, tüm resolver'ların paylaştığı bağımlılık kökü.
type Resolver struct {
	Log *slog.Logger

	// --- kimlik ---
	//
	// Alan adları UC ile bitiyor: gqlgen'in ürettiği resolver metotları da
	// şema alanlarının adını taşıyor (RequestLoginCode gibi), ve aynı adı
	// taşıyan bir alan metodu gölgeler.
	RequestLoginCodeUC *iamUC.RequestLoginCodeUseCase
	VerifyLoginCodeUC  *iamUC.VerifyLoginCodeUseCase
	ManageDevicesUC    *iamUC.ManageDevicesUseCase
	LogoutUC           *iamUC.LogoutUseCase
	Users              iamRepo.UserRepository
	RBAC               iamService.RBACService

	// --- denetim ---
	AuditRepo auditRepo.AuditRepository
}
