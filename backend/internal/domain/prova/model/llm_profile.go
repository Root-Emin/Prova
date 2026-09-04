package model

import (
	"strings"

	"github.com/google/uuid"
)

// Tier, iki LLM kademesinden hangisi.
type Tier string

const (
	// TierFast, her konuşma sırasında karakteri canlandıran kademe.
	// Gecikme burada kullanıcı deneyiminin kendisidir: karakter üç saniye
	// sonra cevap veriyorsa rol yapma hissi kaybolur.
	TierFast Tier = "fast"
	// TierStrong, oturum sonunda rubriğe göre puanlayan kademe. Burada
	// gecikme önemsiz, doğruluk her şey: puan bir sertifikanın dayanağı.
	TierStrong Tier = "strong"
)

// IsValid reports whether the tier is known.
func (t Tier) IsValid() bool { return t == TierFast || t == TierStrong }

// LLMProfile, bir kademenin çalışma anında değiştirilebilir ayarları.
//
// Model adı ve sağlayıcı koda gömülmez, burada durur. Yönetici ekranından
// değiştirildiğinde yeni sürüm oluşur; yeniden derleme ya da yeniden başlatma
// gerekmez.
type LLMProfile struct {
	Document `bson:",inline"`

	Tier     Tier   `bson:"tier" json:"tier"`
	Provider string `bson:"provider" json:"provider"`
	// BaseURL, OpenAI-uyumlu sağlayıcının kökü. Sağlayıcı değiştirmek bu
	// alanı ve model adını değiştirmekten ibaret.
	BaseURL string `bson:"base_url" json:"base_url"`
	Model   string `bson:"model" json:"model"`

	Temperature float64 `bson:"temperature" json:"temperature"`
	TopP        float64 `bson:"top_p" json:"top_p"`
	MaxTokens   int     `bson:"max_tokens" json:"max_tokens"`

	// SystemPromptSuffix, yöneticinin eklediği sistem prompt eki.
	//
	// "Suffix" adı bilinçli: guardrail metni koda gömülü ve prompt derlenirken
	// her zaman en başa gelir. Yönetici buraya ne yazarsa yazsın guardrail'i
	// ezemez, çünkü ondan sonra gelir ve guardrail kendisini ezmeye çalışan
	// talimatları yok saymayı emreder.
	SystemPromptSuffix string `bson:"system_prompt_suffix" json:"system_prompt_suffix"`

	// Birim fiyatlar 1000 token başına. Maliyet hesabı buradan yapılır;
	// koda gömülü bir fiyat listesi ilk zam gününde yanlış olur.
	InputCostPer1K  float64 `bson:"input_cost_per_1k" json:"input_cost_per_1k"`
	OutputCostPer1K float64 `bson:"output_cost_per_1k" json:"output_cost_per_1k"`
}

// Envelope implements the versioned document contract.
func (p *LLMProfile) Envelope() *Document { return &p.Document }

// Validate reports whether the profile can be stored.
func (p *LLMProfile) Validate() error {
	switch {
	case !p.Tier.IsValid():
		return ErrLLMProfileTierInvalid
	case strings.TrimSpace(p.Model) == "":
		return ErrLLMProfileModelRequired
	case strings.TrimSpace(p.BaseURL) == "":
		return ErrLLMProfileBaseURLRequired
	case p.Temperature < 0 || p.Temperature > 2 || p.TopP < 0 || p.TopP > 2:
		return ErrLLMProfileSamplingInvalid
	case p.MaxTokens <= 0:
		return ErrLLMProfileMaxTokensInvalid
	}
	return nil
}

// Cost, verilen token sayıları için maliyeti hesaplar.
func (p *LLMProfile) Cost(inputTokens, outputTokens int) float64 {
	return float64(inputTokens)/1000*p.InputCostPer1K +
		float64(outputTokens)/1000*p.OutputCostPer1K
}

// TemperatureFor, zorluk seviyesinin etkisiyle düzeltilmiş sıcaklık.
//
// Zorluk yalnızca talimatı değil üretim parametresini de değiştirir; ikisinden
// birini atlamak, aynı senaryoda gözle görülür bir fark üretmemek demektir.
func (p *LLMProfile) TemperatureFor(d Difficulty) float64 {
	temperature := p.Temperature + d.TemperatureBias()
	switch {
	case temperature < 0:
		return 0
	case temperature > 2:
		return 2
	default:
		return temperature
	}
}

// NewLLMProfile starts a fresh draft.
func NewLLMProfile(orgID, createdBy uuid.UUID, now nowFunc) *LLMProfile {
	return &LLMProfile{Document: NewLineage(orgID, createdBy, now())}
}
