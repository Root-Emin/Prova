package model

import (
	"strings"

	"github.com/google/uuid"
)

// Criterion, rubriğin tek bir ölçüm maddesi.
type Criterion struct {
	// Key, kriterin makine tarafından okunan kimliği. Puanlama modelinden
	// dönen JSON bu anahtarla eşleşir; başlık değişse bile eşleşme bozulmaz.
	Key         string  `bson:"key" json:"key"`
	Title       string  `bson:"title" json:"title"`
	Description string  `bson:"description" json:"description"`
	Weight      float64 `bson:"weight" json:"weight"`
	MaxPoints   float64 `bson:"max_points" json:"max_points"`
	// Mandatory, düşmesi hâlinde toplam puandan bağımsız olarak KALDI üreten
	// kriter. KVKK'da kişisel veri ifşası tam olarak böyle bir kriterdir:
	// diğer her şeyi mükemmel yapmak onu telafi etmez.
	Mandatory bool `bson:"mandatory" json:"mandatory"`
	// Trap, karaktere talimat olarak verilen tuzak. Doluysa karakter konuşma
	// sırasında bunu deneyerek çalışanı sınar.
	Trap string `bson:"trap,omitempty" json:"trap,omitempty"`
}

// Rubric, oturumun neye göre puanlanacağı.
type Rubric struct {
	Document `bson:",inline"`

	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
	// PassThreshold, geçme eşiği; toplam puanın oranı olarak (0-1).
	PassThreshold float64     `bson:"pass_threshold" json:"pass_threshold"`
	Criteria      []Criterion `bson:"criteria" json:"criteria"`
}

// Envelope implements the versioned document contract.
func (r *Rubric) Envelope() *Document { return &r.Document }

// Validate reports whether the rubric can be stored.
func (r *Rubric) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return ErrRubricNameRequired
	}
	if len(r.Criteria) == 0 {
		return ErrRubricCriteriaRequired
	}
	if r.PassThreshold < 0 || r.PassThreshold > 1 {
		return ErrRubricThresholdInvalid
	}
	// Anahtar çakışması sessizce geçilemez: puanlama modelinden dönen JSON
	// anahtarla eşleşiyor, ve iki kriter aynı anahtarı taşırsa hangisinin
	// puanlandığı belirsizleşir.
	seen := make(map[string]bool, len(r.Criteria))
	for _, c := range r.Criteria {
		key := strings.TrimSpace(c.Key)
		if key == "" || seen[key] {
			return ErrRubricCriterionKeyDup
		}
		seen[key] = true
	}
	return nil
}

// MaxTotal, rubrikten alınabilecek en yüksek ağırlıklı puan.
func (r *Rubric) MaxTotal() float64 {
	var total float64
	for _, c := range r.Criteria {
		total += c.MaxPoints * c.Weight
	}
	return total
}

// MandatoryKeys, zorunlu işaretli kriterlerin anahtarları.
func (r *Rubric) MandatoryKeys() []string {
	keys := make([]string, 0, len(r.Criteria))
	for _, c := range r.Criteria {
		if c.Mandatory {
			keys = append(keys, c.Key)
		}
	}
	return keys
}

// Traps, karaktere talimat olarak verilecek tuzaklar.
func (r *Rubric) Traps() []string {
	traps := make([]string, 0, len(r.Criteria))
	for _, c := range r.Criteria {
		if strings.TrimSpace(c.Trap) != "" {
			traps = append(traps, c.Trap)
		}
	}
	return traps
}

// CriterionByKey returns the criterion with the given key.
func (r *Rubric) CriterionByKey(key string) (Criterion, bool) {
	for _, c := range r.Criteria {
		if c.Key == key {
			return c, true
		}
	}
	return Criterion{}, false
}

// NewRubric starts a fresh draft.
func NewRubric(orgID, createdBy uuid.UUID, now nowFunc) *Rubric {
	return &Rubric{Document: NewLineage(orgID, createdBy, now())}
}
