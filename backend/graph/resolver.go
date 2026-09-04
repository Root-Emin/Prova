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
	provaUC "github.com/masterfabric-go/masterfabric/internal/application/prova/usecase"
	auditRepo "github.com/masterfabric-go/masterfabric/internal/domain/audit/repository"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	iamService "github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/realtime"
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
	VerifyMagicLinkUC  *iamUC.VerifyMagicLinkUseCase
	DeviceChallengeUC  *iamUC.RequestDeviceChallengeUseCase
	AccountUC          *iamUC.AccountUseCase
	Users              iamRepo.UserRepository
	RBAC               iamService.RBACService

	// --- denetim ---
	AuditRepo auditRepo.AuditRepository

	// --- Prova içeriği ve oturumları ---
	//
	// Depo alanları Repo ile bitiyor: gqlgen'in ürettiği resolver metotları
	// şema alan adlarını taşıyor (Scenarios, Characters, Rubrics), ve aynı
	// adı taşıyan bir alan o metodu gölgeler.
	SessionUC      *provaUC.SessionUseCase
	SessionRepo    provaRepo.SessionRepository
	ScoreRepo      provaRepo.ScoreRepository
	CharacterRepo  provaRepo.CharacterRepository
	ScenarioRepo   provaRepo.ScenarioRepository
	RubricRepo     provaRepo.RubricRepository
	LLMProfileRepo provaRepo.LLMProfileRepository
	RoutingRepo    provaRepo.RoutingRepository
	RoutingStatsUC *provaUC.RoutingStatsUseCase

	// İçerik yazma akışları. Generic ContentUseCase, dört belge türünün
	// paylaştığı oluştur/düzenle/yayınla üçlüsünü tek yerde tutuyor.
	CharacterUC  *provaUC.ContentUseCase[provaModel.Character, *provaModel.Character]
	ScenarioUC   *provaUC.ContentUseCase[provaModel.Scenario, *provaModel.Scenario]
	RubricUC     *provaUC.ContentUseCase[provaModel.Rubric, *provaModel.Rubric]
	LLMProfileUC *provaUC.LLMProfileUseCase
	Broker       *realtime.SessionBroker
}
