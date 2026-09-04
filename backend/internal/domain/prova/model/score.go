package model

import (
	"time"

	"github.com/google/uuid"
)

// CriterionScore, tek bir kriterin puanı ve gerekçesi.
type CriterionScore struct {
	CriterionKey string  `bson:"criterion_key" json:"criterion_key"`
	Title        string  `bson:"title" json:"title"`
	Weight       float64 `bson:"weight" json:"weight"`
	Mandatory    bool    `bson:"mandatory" json:"mandatory"`
	Points       float64 `bson:"points" json:"points"`
	MaxPoints    float64 `bson:"max_points" json:"max_points"`
	Rationale    string  `bson:"rationale" json:"rationale"`
	// Quote, modelin gerekçe olarak gösterdiği transkript alıntısı.
	Quote string `bson:"quote" json:"quote"`
	// TurnIndex, alıntının geldiğini iddia ettiği konuşma sırası.
	TurnIndex *int `bson:"turn_index,omitempty" json:"turn_index,omitempty"`
	// QuoteVerified, alıntının o sıranın metninde gerçekten bulunup
	// bulunmadığı.
	//
	// Uydurma alıntı ürünün güvenilirliğini bitirir: "şunu söylediniz" diyen
	// bir sertifika, söylenmemiş bir cümleyi gösterirse tüm puanlama
	// tartışmalı hâle gelir. Bu yüzden sessizce geçilmiyor, işaretleniyor.
	QuoteVerified bool `bson:"quote_verified" json:"quote_verified"`
}

// ScoreOverride, yöneticinin puanı ezme kaydı.
//
// Ezme, puanı değiştirmez — üzerine yazar ve eskisini saklar. Ezilmiş bir
// puanın önceki hâli görülemezse, ezme yetkisi denetlenemez bir yetkiye
// dönüşür.
type ScoreOverride struct {
	OverriddenBy  uuid.UUID `bson:"overridden_by" json:"overridden_by"`
	Reason        string    `bson:"reason" json:"reason"`
	PreviousTotal float64   `bson:"previous_total" json:"previous_total"`
	PreviousPass  bool      `bson:"previous_passed" json:"previous_passed"`
	OverriddenAt  time.Time `bson:"overridden_at" json:"overridden_at"`
}

// Score, oturumun puanlanmış sonucu.
type Score struct {
	ID        uuid.UUID `bson:"_id" json:"id"`
	OrgID     uuid.UUID `bson:"org_id" json:"org_id"`
	SessionID uuid.UUID `bson:"session_id" json:"session_id"`

	Total     float64 `bson:"total" json:"total"`
	MaxTotal  float64 `bson:"max_total" json:"max_total"`
	Passed    bool    `bson:"passed" json:"passed"`
	Threshold float64 `bson:"threshold" json:"threshold"`

	// FailedMandatoryKeys, düşen zorunlu kriterler. Doluysa toplam puan ne
	// olursa olsun Passed false'tur.
	FailedMandatoryKeys []string         `bson:"failed_mandatory_keys" json:"failed_mandatory_keys"`
	Criteria            []CriterionScore `bson:"criteria" json:"criteria"`

	// Model, puanı üreten model kimliği. Puanlama modelinin değişmesi
	// sonuçları da değiştirir; hangi modelin puanladığı kayıtta olmalı.
	Model string `bson:"model" json:"model"`
	// RubricRef, puanlamada kullanılan rubrik sürümü.
	RubricRef Reference `bson:"rubric_ref" json:"rubric_ref"`

	Override  *ScoreOverride `bson:"override,omitempty" json:"override,omitempty"`
	CreatedAt time.Time      `bson:"created_at" json:"created_at"`
}

// Evaluate, kriter puanlarından toplamı ve geçme kararını hesaplar.
//
// Karar iki kuraldan geçer ve ikincisi birincisini ezer: ağırlıklı toplam
// eşiği aşmalı VE hiçbir zorunlu kriter düşmemeli. Zorunlu kriter kuralı
// telafi edilemez olmalı, yoksa "kişisel veri ifşa etme" maddesi diğer
// maddelerden toplanan puanla satın alınabilir hâle gelir.
func (s *Score) Evaluate(threshold float64) {
	s.Threshold = threshold
	s.Total = 0
	s.MaxTotal = 0
	s.FailedMandatoryKeys = nil

	for _, c := range s.Criteria {
		s.Total += c.Points * c.Weight
		s.MaxTotal += c.MaxPoints * c.Weight
		if c.Mandatory && c.Points < c.MaxPoints {
			s.FailedMandatoryKeys = append(s.FailedMandatoryKeys, c.CriterionKey)
		}
	}

	ratio := 0.0
	if s.MaxTotal > 0 {
		ratio = s.Total / s.MaxTotal
	}
	s.Passed = ratio >= threshold && len(s.FailedMandatoryKeys) == 0
}

// UnverifiedQuoteCount, transkriptte bulunamayan alıntı sayısı.
func (s *Score) UnverifiedQuoteCount() int {
	var count int
	for _, c := range s.Criteria {
		if !c.QuoteVerified {
			count++
		}
	}
	return count
}
