package service

import (
	"context"
	"time"

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

	breakers *breakers
}

// RouterConfig, yönlendiricinin ayarları.
type RouterConfig struct {
	LongInputThreshold int
	// CircuitThreshold, devrenin açılması için gereken ardışık hata sayısı.
	CircuitThreshold int
	// CircuitCooldown, devre açıldıktan sonra tek deneme yapılana kadar
	// geçen süre.
	CircuitCooldown time.Duration
	// Now, test edilebilirlik için saat kaynağı. Boşsa time.Now.
	Now func() time.Time
}

// NewRuleRouter wires the router with its circuit breakers.
func NewRuleRouter(cfg RouterConfig) *RuleRouter {
	threshold := cfg.LongInputThreshold
	if threshold <= 0 {
		threshold = defaultLongInputThreshold
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &RuleRouter{
		LongInputThreshold: threshold,
		breakers:           newBreakers(cfg.CircuitThreshold, cfg.CircuitCooldown, now),
	}
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

	// 3. Hızlı kademenin devresi açıksa güçlüye. Kural sırası burada
	//    önemli: uzun girdi kuralından ÖNCE geliyor, çünkü devre açıkken
	//    girdinin uzunluğunun bir önemi yok — hızlı kademe zaten
	//    kullanılamaz durumda ve denemek yalnızca gecikme ekler.
	if !r.breakers.fast.allow() {
		return model.TierStrong, model.RuleFailover
	}

	// 4. Girdi çok uzunsa güçlüye.
	if in.InputLength > r.LongInputThreshold {
		return model.TierStrong, model.RuleLongInput
	}

	// 5. Diğer her durum hızlıya: konuşma sırasının olağan yolu, ve
	//    tasarrufun tamamı buradan geliyor.
	return model.TierFast, model.RuleDefault
}

// ReportResult implements Router.
func (r *RuleRouter) ReportResult(tier model.Tier, success bool) {
	r.breakers.forTier(tier).report(success)
}

// IsOpen implements Router.
func (r *RuleRouter) IsOpen(tier model.Tier) bool {
	return r.breakers.forTier(tier).isOpen()
}

var _ Router = (*RuleRouter)(nil)
