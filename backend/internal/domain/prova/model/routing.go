package model

import (
	"time"

	"github.com/google/uuid"
)

// RoutingRule, bir çağrının hangi kurala göre yönlendirildiği.
//
// Sabit isimler kullanılıyor çünkü yönlendirme istatistiği bunlara göre
// gruplanıyor: serbest metin bir kural adı, "hangi kural ne sıklıkta
// tetikleniyor" sorusunu cevaplanamaz hâle getirir.
type RoutingRule string

const (
	// RuleScoring, oturum sonu puanlaması. Her zaman güçlü kademeye gider.
	RuleScoring RoutingRule = "scoring"
	// RuleMandatorySignal, zorunlu kriter sinyali tetiklendi. Kararın
	// sonucu KALDI olabilir; bu kararı hızlı modele bırakmak riskli.
	RuleMandatorySignal RoutingRule = "mandatory_signal"
	// RuleFailover, hızlı kademenin devresi açık. Oturum kesintiye
	// uğramasın diye güçlü kademe devralır.
	RuleFailover RoutingRule = "failover"
	// RuleLongInput, girdi hızlı modelin iyi taşıyacağı uzunluğu aştı.
	RuleLongInput RoutingRule = "long_input"
	// RuleDefault, diğer her durum. Konuşma sırasının olağan yolu.
	RuleDefault RoutingRule = "default"
)

// RoutingRecord, tek bir LLM çağrısının yönlendirme kararı ve sonucu.
//
// Bu koleksiyon, otomatik dağılım iddiasının tek kanıtı. Grafik çizilebilir
// veri üretmiyorsa iddia doğrulanamaz; bu yüzden gecikme, token sayıları ve
// maliyet çağrı başına kaydediliyor, özet olarak değil.
type RoutingRecord struct {
	ID    uuid.UUID `bson:"_id" json:"id"`
	OrgID uuid.UUID `bson:"org_id" json:"org_id"`
	// SessionID, çağrının ait olduğu oturum. Profil testi gibi oturum dışı
	// çağrılarda boştur.
	SessionID *uuid.UUID `bson:"session_id,omitempty" json:"session_id,omitempty"`

	Tier Tier        `bson:"tier" json:"tier"`
	Rule RoutingRule `bson:"rule" json:"rule"`
	// Model, çağrılan model kimliği. Profil sürümlendiği için burada da
	// saklanıyor: eski kayıtlar, o an hangi modelin kullanıldığını
	// göstermeye devam etmeli.
	Model     string    `bson:"model" json:"model"`
	ProfileID uuid.UUID `bson:"profile_id" json:"profile_id"`

	LatencyMs    int     `bson:"latency_ms" json:"latency_ms"`
	InputTokens  int     `bson:"input_tokens" json:"input_tokens"`
	OutputTokens int     `bson:"output_tokens" json:"output_tokens"`
	CostUSD      float64 `bson:"cost_usd" json:"cost_usd"`

	Success bool   `bson:"success" json:"success"`
	Error   string `bson:"error,omitempty" json:"error,omitempty"`

	// MaskedFieldCount, bu çağrıda maskelenen kişisel veri alanı sayısı.
	MaskedFieldCount int `bson:"masked_field_count" json:"masked_field_count"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// TierStats, bir kademenin toplu istatistiği.
type TierStats struct {
	Tier         Tier    `json:"tier"`
	Calls        int     `json:"calls"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	CostUSD      float64 `json:"cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
}

// RoutingStats, dağılımın grafiğe dökülebilir özeti.
type RoutingStats struct {
	PerTier          []TierStats `json:"per_tier"`
	FailoverCount    int         `json:"failover_count"`
	TotalCostUSD     float64     `json:"total_cost_usd"`
	AllStrongCostUSD float64     `json:"all_strong_cost_usd"`
	SavingsPercent   float64     `json:"savings_percent"`
}

// ComputeSavings, tasarruf yüzdesini doldurur.
//
// Ayrı bir metot: "hepsi güçlü kademeye gitseydi" maliyeti kayıtlardan değil
// güçlü profilin birim fiyatından hesaplanır, ve o hesabı yapan katman bu
// yapıyı doldurduktan sonra çağırır.
func (s *RoutingStats) ComputeSavings() {
	if s.AllStrongCostUSD <= 0 {
		s.SavingsPercent = 0
		return
	}
	saved := s.AllStrongCostUSD - s.TotalCostUSD
	s.SavingsPercent = saved / s.AllStrongCostUSD * 100
}
