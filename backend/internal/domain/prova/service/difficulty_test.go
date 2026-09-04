package service

import (
	"strings"
	"testing"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// Faz 5'in kontrol maddesi: aynı senaryoda zorluk değiştirildiğinde
// karakterin davranışı belirgin şekilde farklı olmalı. Bunun promptta
// karşılığı, aynı senaryo ve karakter metniyle derlenen iki promptun farklı
// davranış talimatı taşıması.
func TestCompileCharacterPrompt_SameScenarioDifferentDifficultyProducesDifferentPrompts(t *testing.T) {
	character := &model.Character{
		Persona:       "Bir müşterisin.",
		BehaviorRules: []string{"Kısa konuşursun."},
	}
	scenario := &model.Scenario{Context: "Çağrı merkezi"}
	rubric := &model.Rubric{Criteria: []model.Criterion{{Key: "tone", Title: "Ton"}}}

	build := func(d model.Difficulty) string {
		character.Difficulty = d
		messages := BuildCharacterMessages(CharacterTurnInput{
			Character: character, Scenario: scenario, Rubric: rubric,
			EmployeeMessage: "Merhaba",
		})
		return messages[0].Content
	}

	low := build(model.DifficultyLow)
	high := build(model.DifficultyHigh)

	if low == high {
		t.Fatal("zorluk değiştiğinde prompt da değişmeli")
	}
	if !strings.Contains(low, "İşbirlikçi") {
		t.Error("düşük zorluk promptu işbirlikçi davranış istemeli")
	}
	if !strings.Contains(high, "Dirençli") {
		t.Error("yüksek zorluk promptu dirençli davranış istemeli")
	}
	// Guardrail her iki promptta da aynı ve en başta kalmalı: zorluk
	// karakterin davranışını değiştirir, kurallarını değil.
	if !strings.HasPrefix(low, Guardrail) || !strings.HasPrefix(high, Guardrail) {
		t.Error("guardrail her zorluk seviyesinde en başta kalmalı")
	}
}
