package model

import (
	"strings"

	"github.com/google/uuid"
)

// Difficulty, karakterin ne kadar dirençli davranacağını belirler.
//
// Yalnızca bir etiket değil: hem üretim parametrelerini (sıcaklık) hem de
// prompt'a giren davranış talimatını etkiler. İkisinden yalnızca birini
// değiştirmek, aynı senaryoda gözle görülür bir fark üretmezdi.
type Difficulty string

const (
	DifficultyLow     Difficulty = "low"
	DifficultyMedium  Difficulty = "medium"
	DifficultyHigh    Difficulty = "high"
	DifficultyExtreme Difficulty = "extreme"
)

// IsValid reports whether the value is one of the known levels.
func (d Difficulty) IsValid() bool {
	switch d {
	case DifficultyLow, DifficultyMedium, DifficultyHigh, DifficultyExtreme:
		return true
	default:
		return false
	}
}

// TemperatureBias, zorluk seviyesinin üretim sıcaklığına katkısı.
//
// Yüksek zorlukta sıcaklık artar: dirençli bir karakter tahmin edilebilir
// olmamalı, aksi hâlde çalışan iki oturum sonra ezberler ve sınav ölçmeyi
// bırakır.
func (d Difficulty) TemperatureBias() float64 {
	switch d {
	case DifficultyLow:
		return -0.15
	case DifficultyHigh:
		return 0.10
	case DifficultyExtreme:
		return 0.20
	default:
		return 0
	}
}

// BehaviourInstruction, zorluk seviyesinin prompt'a giren karşılığı.
//
// Metin koda gömülü: yöneticinin değiştirebildiği alanlar karakterin kendi
// tanımı ve davranış kuralları. Zorluğun ne anlama geldiği ürünün ölçüm
// birimidir; organizasyondan organizasyona değişirse iki sertifika
// karşılaştırılamaz hâle gelir.
func (d Difficulty) BehaviourInstruction() string {
	switch d {
	case DifficultyLow:
		return "İşbirlikçi davran. Çalışan doğru yöne gittiğinde onayla, " +
			"sorduğu bilgiyi fazla direnmeden ver, tonun sakin kalsın."
	case DifficultyMedium:
		return "Ölçülü davran. Bilgiyi hemen verme, çalışan doğru soruyu " +
			"sorduğunda ver. Bir kez itiraz et, ikna edilirsen kabul et."
	case DifficultyHigh:
		return "Dirençli davran. Sabırsız ve şüphecisin. Bilgiyi ancak " +
			"çalışan gerekçesini açıkladığında ver, en az iki kez itiraz et, " +
			"konuyu dağıtmaya çalış."
	case DifficultyExtreme:
		return "Çok dirençli davran. Öfkeli ve ısrarcısın, sözü kesersin, " +
			"kuralların dışına çıkılmasını talep edersin. Çalışan prosedürü " +
			"net biçimde savunmadıkça hiçbir şeyi kabul etme."
	default:
		return ""
	}
}

// Character, çalışanın karşısına çıkan kişi.
//
// Karakter ve senaryo ayrı belgeler: aynı "sabırsız müşteri" birden fazla
// senaryoda oynatılabilmeli, ve karakterin zorluğunu değiştirmek senaryonun
// sürümünü artırmamalı.
type Character struct {
	Document `bson:",inline"`

	Name string `bson:"name" json:"name"`
	// Persona, karakterin kim olduğu. Sistem prompt'una olduğu gibi girer.
	Persona string `bson:"persona" json:"persona"`
	// BehaviorRules, karakterin uyacağı davranış kuralları.
	BehaviorRules []string   `bson:"behavior_rules" json:"behavior_rules"`
	Difficulty    Difficulty `bson:"difficulty" json:"difficulty"`
	// HiddenFacts, karakterin kendiliğinden söylemeyeceği bilgiler. Çalışan
	// doğru soruyu sorarsa açılır; sınavın ölçtüğü şeylerden biri budur.
	HiddenFacts []string `bson:"hidden_facts" json:"hidden_facts"`
}

// Envelope implements the versioned document contract.
func (c *Character) Envelope() *Document { return &c.Document }

// Validate reports whether the character can be stored.
func (c *Character) Validate() error {
	switch {
	case strings.TrimSpace(c.Name) == "":
		return ErrCharacterNameRequired
	case strings.TrimSpace(c.Persona) == "":
		return ErrCharacterPersonaRequired
	case !c.Difficulty.IsValid():
		return ErrCharacterDifficultyInvalid
	}
	return nil
}

// NewCharacter starts a fresh draft.
func NewCharacter(orgID, createdBy uuid.UUID, now nowFunc) *Character {
	return &Character{Document: NewLineage(orgID, createdBy, now())}
}
