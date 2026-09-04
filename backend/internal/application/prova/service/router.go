package service

import (
	"context"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// RuleRouter, kural tabanlı kademe yönlendiricisi.
//
// Kurallar sırayla değerlendirilir ve ilk eşleşen kazanır. Sıra keyfi değil:
// en pahalı ama en gerekli karar (puanlama) başta, en ucuz varsayılan sonda.
type RuleRouter struct {
	// LongInputThreshold, bu karakter sayısını aşan girdiyi güçlü kademeye
	// yönlendirir. Hızlı modeller uzun bağlamda gözle görülür şekilde bozulur
	// ve o bozulma karakterin tutarsızlaşması olarak görünür.
	LongInputThreshold int
}

// NewRuleRouter wires the router.
func NewRuleRouter(longInputThreshold int) *RuleRouter {
	if longInputThreshold <= 0 {
		longInputThreshold = defaultLongInputThreshold
	}
	return &RuleRouter{LongInputThreshold: longInputThreshold}
}

// defaultLongInputThreshold, yapılandırma verilmediğinde uygulanan eşik.
const defaultLongInputThreshold = 4000

// Route implements Router.
func (r *RuleRouter) Route(_ context.Context, in RouteInput) (model.Tier, model.RoutingRule) {
	// 1. Oturum sonu puanlaması her zaman güçlü kademeye. Puan bir
	//    sertifikanın dayanağı; hızlı modelin tasarrufu bunu riske atmaya
	//    değmez.
	if in.Purpose == PurposeScoring {
		return model.TierStrong, model.RuleScoring
	}

	// 2. Zorunlu kriter sinyali tetiklendiyse güçlüye. Bu kararın sonucu
	//    doğrudan KALDI olabilir, ve o kararı hızlı modele bırakmak, ürünün
	//    en ağır sonucunu en zayıf modele emanet etmek olurdu.
	if in.MandatorySignal {
		return model.TierStrong, model.RuleMandatorySignal
	}

	// 3. Girdi çok uzunsa güçlüye.
	if in.InputLength > r.LongInputThreshold {
		return model.TierStrong, model.RuleLongInput
	}

	// 4. Diğer her durum hızlıya: konuşma sırasının olağan yolu, ve
	//    tasarrufun tamamı buradan geliyor.
	return model.TierFast, model.RuleDefault
}

// ReportResult implements Router. Devre kesici Faz 4'te bu sayaca bağlanıyor.
func (r *RuleRouter) ReportResult(model.Tier, bool) {}

// IsOpen implements Router.
func (r *RuleRouter) IsOpen(model.Tier) bool { return false }

var _ Router = (*RuleRouter)(nil)
