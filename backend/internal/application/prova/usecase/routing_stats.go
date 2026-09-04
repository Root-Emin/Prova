package usecase

import (
	"context"
	"errors"
	"time"

	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// RoutingStatsUseCase, dağılım istatistiğini üretir.
//
// Bu sorgu, otomatik dağılım iddiasının tek kanıtı: grafik çizilebilir veri
// üretmiyorsa iddia doğrulanamaz. Bu yüzden kademe başına çağrı, gecikme ve
// maliyetin yanında "hepsi güçlü kademeye gitseydi" karşılaştırması da
// dönüyor — tasarruf yüzdesi olmadan dağılımın işe yarayıp yaramadığı
// söylenemez.
type RoutingStatsUseCase struct {
	routing  provaRepo.RoutingRepository
	profiles provaRepo.LLMProfileRepository
}

// NewRoutingStatsUseCase wires the use case.
func NewRoutingStatsUseCase(routing provaRepo.RoutingRepository, profiles provaRepo.LLMProfileRepository) *RoutingStatsUseCase {
	return &RoutingStatsUseCase{routing: routing, profiles: profiles}
}

// Execute returns the aggregate for the window.
func (uc *RoutingStatsUseCase) Execute(ctx context.Context, scope provaRepo.Scope, from, to *time.Time) (*provaModel.RoutingStats, error) {
	perTier, failovers, err := uc.routing.Stats(ctx, scope, from, to)
	if err != nil {
		return nil, err
	}

	stats := &provaModel.RoutingStats{
		PerTier:       perTier,
		FailoverCount: failovers,
	}

	var totalInput, totalOutput int
	for _, tier := range perTier {
		stats.TotalCostUSD += tier.CostUSD
		totalInput += tier.InputTokens
		totalOutput += tier.OutputTokens
	}

	// Karşılaştırma maliyeti, güçlü profilin BUGÜNKÜ birim fiyatlarından
	// hesaplanıyor. Kayıtlardaki tarihsel fiyatları kullanmak daha doğru
	// olurdu ama karşılaştırma "bugün hepsini güçlüye gönderseydik ne
	// öderdik" sorusunu cevaplıyor, "geçmişte ne öderdik" sorusunu değil.
	strong, err := uc.profiles.GetPublishedByTier(ctx, scope, provaModel.TierStrong)
	switch {
	case err == nil:
		stats.AllStrongCostUSD = strong.Cost(totalInput, totalOutput)
	case errors.Is(err, domainErr.ErrNotFound):
		// Güçlü profil yoksa karşılaştırma yapılamaz. Sıfır bırakmak
		// tasarrufu %0 gösterir; bu, "veri yok" için doğru cevap, çünkü
		// uydurulmuş bir karşılaştırma yanlış bir iddia olurdu.
		stats.AllStrongCostUSD = 0
	default:
		return nil, err
	}

	stats.ComputeSavings()
	return stats, nil
}
