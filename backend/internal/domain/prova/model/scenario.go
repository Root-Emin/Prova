package model

import (
	"strings"

	"github.com/google/uuid"
)

// Scenario, oynanacak durum.
//
// Karaktere ve rubriğe sürüm referansıyla bağlanır, soy kimliğiyle değil.
// Yalnızca soyu tutan bir bağ, sonradan yayınlanan sürüme sessizce kayar ve
// senaryonun neye karşı oynandığı sonradan değişmiş olurdu.
type Scenario struct {
	Document `bson:",inline"`

	Title     string `bson:"title" json:"title"`
	Context   string `bson:"context" json:"context"`
	Objective string `bson:"objective" json:"objective"`

	CharacterRef Reference `bson:"character_ref" json:"character_ref"`
	RubricRef    Reference `bson:"rubric_ref" json:"rubric_ref"`

	// MaxTurns, oturumun kaç konuşma sırasında biteceği. Sınırsız bir oturum
	// hem maliyeti hem de bağlam penceresini kontrolsüz bırakır.
	MaxTurns int `bson:"max_turns" json:"max_turns"`
}

// Envelope implements the versioned document contract.
func (s *Scenario) Envelope() *Document { return &s.Document }

// Validate reports whether the scenario can be stored.
func (s *Scenario) Validate() error {
	switch {
	case strings.TrimSpace(s.Title) == "":
		return ErrScenarioTitleRequired
	case strings.TrimSpace(s.Context) == "":
		return ErrScenarioContextRequired
	case s.CharacterRef.VersionID == uuid.Nil:
		return ErrScenarioCharacterRequired
	case s.RubricRef.VersionID == uuid.Nil:
		return ErrScenarioRubricRequired
	}
	return nil
}

// EffectiveMaxTurns, sıfır bırakılmış limiti makul bir varsayılana çeker.
func (s *Scenario) EffectiveMaxTurns() int {
	if s.MaxTurns <= 0 {
		return defaultMaxTurns
	}
	return s.MaxTurns
}

// defaultMaxTurns, senaryo bir limit vermediğinde uygulanan tavan.
const defaultMaxTurns = 20

// NewScenario starts a fresh draft.
func NewScenario(orgID, createdBy uuid.UUID, now nowFunc) *Scenario {
	return &Scenario{Document: NewLineage(orgID, createdBy, now())}
}
