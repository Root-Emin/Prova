package service

import (
	"testing"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

func transcript() []*model.Turn {
	return []*model.Turn{
		{Index: 0, Role: model.TurnRoleCharacter, Text: "Eşimin hesabındaki işlemi öğrenmek istiyorum."},
		{Index: 1, Role: model.TurnRoleEmployee, Text: "Maalesef hesap sahibi olmayan kişilere bilgi veremiyorum."},
		{Index: 2, Role: model.TurnRoleCharacter, Text: "Ama ben eşiyim! Avukatımı arayacağım."},
		{Index: 3, Role: model.TurnRoleEmployee, Text: "Anlıyorum, ancak vekâletname ile şubeden başvurmanız gerekiyor."},
	}
}

func turnPtr(i int) *int { return &i }

func TestQuoteVerifier_AcceptsAnExactQuoteFromTheClaimedTurn(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	verified, found := v.Verify("hesap sahibi olmayan kişilere bilgi veremiyorum", turnPtr(1))

	if !verified {
		t.Fatal("transkriptte gerçekten olan alıntı doğrulanmalı")
	}
	if found == nil || *found != 1 {
		t.Fatalf("bulunan sıra 1 olmalı: %v", found)
	}
}

// Asıl korunan şey bu: model uydurduğu bir cümleyi gerekçe olarak
// gösterirse, puan sessizce kabul edilmemeli.
func TestQuoteVerifier_RejectsAFabricatedQuote(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	verified, _ := v.Verify("Tabii, hesap hareketlerini hemen okuyorum.", turnPtr(1))

	if verified {
		t.Fatal("transkriptte olmayan alıntı doğrulanmamalı")
	}
}

// Alıntı doğru ama sıra numarası yanlışsa iddia yine yanlıştır: model hem
// alıntıyı hem de nereden geldiğini iddia ediyor.
func TestQuoteVerifier_RejectsARealQuoteAttributedToTheWrongTurn(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	verified, _ := v.Verify("hesap sahibi olmayan kişilere bilgi veremiyorum", turnPtr(3))

	if verified {
		t.Fatal("yanlış sıraya atfedilen alıntı doğrulanmamalı")
	}
}

// Sıra numarası vermeyen bir model cezalandırılmamalı; tüm transkript
// taranır ve bulunan sıra doldurulur.
func TestQuoteVerifier_ScansEveryTurnWhenNoTurnIsClaimed(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	verified, found := v.Verify("vekâletname ile şubeden başvurmanız", nil)

	if !verified {
		t.Fatal("sıra numarası verilmeyen gerçek alıntı bulunmalı")
	}
	if found == nil || *found != 3 {
		t.Fatalf("bulunan sıra 3 olmalı: %v", found)
	}
}

// Noktalama ve büyük/küçük harf farkları yanlış negatif üretmemeli: yanlış
// negatifler doğrulamanın kendisine olan güveni yok eder.
func TestQuoteVerifier_IgnoresPunctuationAndCase(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	cases := []string{
		"MAALESEF HESAP SAHİBİ OLMAYAN KİŞİLERE BİLGİ VEREMİYORUM",
		"maalesef, hesap sahibi olmayan kişilere bilgi veremiyorum!",
		"«maalesef hesap sahibi olmayan   kişilere bilgi veremiyorum»",
	}
	for _, quote := range cases {
		if verified, _ := v.Verify(quote, turnPtr(1)); !verified {
			t.Errorf("normalize edildiğinde eşleşmeliydi: %q", quote)
		}
	}
}

// Boş alıntı doğrulanmış sayılmamalı: gerekçesiz bir puan, gerekçesi
// uydurulmuş bir puandan daha iyi değil.
func TestQuoteVerifier_TreatsAnEmptyQuoteAsUnverified(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	for _, quote := range []string{"", "   ", "..."} {
		if verified, _ := v.Verify(quote, turnPtr(1)); verified {
			t.Errorf("boş alıntı doğrulanmamalı: %q", quote)
		}
	}
}

// Var olmayan bir sıra numarası, doğrulamayı çökertmemeli.
func TestQuoteVerifier_HandlesAnOutOfRangeTurnIndex(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	if verified, _ := v.Verify("herhangi bir şey", turnPtr(99)); verified {
		t.Fatal("var olmayan sıra doğrulanmamalı")
	}
}

func TestQuoteVerifier_VerifyAllMarksEveryCriterion(t *testing.T) {
	v := NewQuoteVerifier(transcript())

	criteria := []model.CriterionScore{
		{CriterionKey: "gercek", Quote: "vekâletname ile şubeden başvurmanız", TurnIndex: turnPtr(3)},
		{CriterionKey: "uydurma", Quote: "hemen okuyorum efendim", TurnIndex: turnPtr(1)},
		{CriterionKey: "sirasiz", Quote: "avukatımı arayacağım"},
	}

	result := v.VerifyAll(criteria)

	if !result[0].QuoteVerified {
		t.Error("gerçek alıntı doğrulanmalı")
	}
	if result[1].QuoteVerified {
		t.Error("uydurma alıntı doğrulanmamalı")
	}
	if !result[2].QuoteVerified {
		t.Error("sıra numarasız gerçek alıntı doğrulanmalı")
	}
	if result[2].TurnIndex == nil || *result[2].TurnIndex != 2 {
		t.Errorf("bulunan sıra doldurulmalı: %v", result[2].TurnIndex)
	}
}

// Türkçe'nin "İ" harfi unicode.ToLower ile birleşen noktalı bir "i"ye
// dönüşür ve kullanıcının yazdığı düz "i" ile eşleşmez. Bu, doğrulamayı
// Türkçe metinlerde sessizce bozan türden bir ayrıntı.
func TestNormalizeForQuote_HandlesTurkishDottedCapitalI(t *testing.T) {
	if got, want := NormalizeForQuote("İSTANBUL"), NormalizeForQuote("istanbul"); got != want {
		t.Fatalf("İ ve i normalize edildiğinde eşleşmeli: %q != %q", got, want)
	}
}
