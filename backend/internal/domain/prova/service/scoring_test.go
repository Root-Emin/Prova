package service

import (
	"strings"
	"testing"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

func testRubric() *model.Rubric {
	return &model.Rubric{
		Name:          "Test",
		PassThreshold: 0.7,
		Criteria: []model.Criterion{
			{Key: "pii", Title: "Kişisel veri", Weight: 2, MaxPoints: 5, Mandatory: true},
			{Key: "tone", Title: "Ton", Weight: 1, MaxPoints: 5},
			{Key: "closing", Title: "Kapanış", Weight: 1, MaxPoints: 5},
		},
	}
}

func TestParseScoring_MapsModelOutputOntoRubricCriteria(t *testing.T) {
	raw := `{"criteria":[
		{"key":"pii","points":5,"rationale":"Bilgi vermedi","quote":"bilgi veremiyorum","turn_index":1},
		{"key":"tone","points":4,"rationale":"Sakin kaldı","quote":"anlıyorum","turn_index":3},
		{"key":"closing","points":3,"rationale":"Kısa kapattı","quote":"iyi günler","turn_index":5}
	]}`

	scores, err := ParseScoring(raw, testRubric())

	if err != nil {
		t.Fatalf("çözümleme başarısız: %v", err)
	}
	if len(scores) != 3 {
		t.Fatalf("üç kriter beklenir, %d geldi", len(scores))
	}
	if scores[0].CriterionKey != "pii" || scores[0].Points != 5 || !scores[0].Mandatory {
		t.Fatalf("zorunlu kriter doğru eşlenmedi: %+v", scores[0])
	}
	if scores[0].TurnIndex == nil || *scores[0].TurnIndex != 1 {
		t.Fatal("sıra numarası taşınmalı")
	}
}

// Model bir kriteri atlarsa o kriter sessizce kaybolmamalı: aksi hâlde
// zorunlu bir kriteri atlamak, KALDI kuralını atlamanın yolu olurdu.
func TestParseScoring_FillsInCriteriaTheModelSkipped(t *testing.T) {
	raw := `{"criteria":[{"key":"tone","points":5,"rationale":"iyi","quote":"anlıyorum","turn_index":1}]}`

	scores, err := ParseScoring(raw, testRubric())

	if err != nil {
		t.Fatalf("çözümleme başarısız: %v", err)
	}
	if len(scores) != 3 {
		t.Fatalf("rubrikteki her kriter dönmeli, %d geldi", len(scores))
	}
	var pii model.CriterionScore
	for _, s := range scores {
		if s.CriterionKey == "pii" {
			pii = s
		}
	}
	if pii.Points != 0 {
		t.Fatalf("atlanan kriter sıfır puan almalı, %v geldi", pii.Points)
	}
	if !strings.Contains(pii.Rationale, "döndürmedi") {
		t.Fatalf("atlanan kriterin gerekçesi açık olmalı: %q", pii.Rationale)
	}
}

// Aralık dışı puan geçme eşiğini anlamsızlaştırır.
func TestParseScoring_ClampsOutOfRangePoints(t *testing.T) {
	raw := `{"criteria":[
		{"key":"pii","points":12,"quote":"x"},
		{"key":"tone","points":-3,"quote":"y"},
		{"key":"closing","points":2,"quote":"z"}
	]}`

	scores, _ := ParseScoring(raw, testRubric())

	if scores[0].Points != 5 {
		t.Errorf("üst sınır uygulanmalı: %v", scores[0].Points)
	}
	if scores[1].Points != 0 {
		t.Errorf("alt sınır uygulanmalı: %v", scores[1].Points)
	}
}

func TestParseScoring_AcceptsJSONWrappedInACodeFence(t *testing.T) {
	raw := "```json\n{\"criteria\":[{\"key\":\"pii\",\"points\":5,\"quote\":\"x\"}]}\n```"

	if _, err := ParseScoring(raw, testRubric()); err != nil {
		t.Fatalf("kod bloğuna sarılmış JSON kabul edilmeli: %v", err)
	}
}

// Faz 3'ün kontrol maddesi: zorunlu kriter düşünce sonuç KALDI.
func TestScore_MandatoryCriterionFailureForcesFailure(t *testing.T) {
	score := &model.Score{Criteria: []model.CriterionScore{
		// Zorunlu kriter düşük; diğer her şey tam.
		{CriterionKey: "pii", Weight: 2, MaxPoints: 5, Points: 2, Mandatory: true},
		{CriterionKey: "tone", Weight: 1, MaxPoints: 5, Points: 5},
		{CriterionKey: "closing", Weight: 1, MaxPoints: 5, Points: 5},
	}}

	score.Evaluate(0.5)

	ratio := score.Total / score.MaxTotal
	if ratio < 0.5 {
		t.Fatalf("bu test, eşiği AŞAN bir toplamla anlamlı: oran %v", ratio)
	}
	if score.Passed {
		t.Fatal("zorunlu kriter düştüğünde toplam puan ne olursa olsun sonuç KALDI olmalı")
	}
	if len(score.FailedMandatoryKeys) != 1 || score.FailedMandatoryKeys[0] != "pii" {
		t.Fatalf("düşen zorunlu kriter kayda geçmeli: %v", score.FailedMandatoryKeys)
	}
}

func TestScore_PassesWhenThresholdMetAndNoMandatoryFailure(t *testing.T) {
	score := &model.Score{Criteria: []model.CriterionScore{
		{CriterionKey: "pii", Weight: 2, MaxPoints: 5, Points: 5, Mandatory: true},
		{CriterionKey: "tone", Weight: 1, MaxPoints: 5, Points: 4},
		{CriterionKey: "closing", Weight: 1, MaxPoints: 5, Points: 3},
	}}

	score.Evaluate(0.7)

	if !score.Passed {
		t.Fatalf("eşik aşıldı ve zorunlu kriter düşmedi; geçmeliydi (%v/%v)", score.Total, score.MaxTotal)
	}
	if len(score.FailedMandatoryKeys) != 0 {
		t.Fatal("düşen zorunlu kriter olmamalı")
	}
}

func TestScore_FailsWhenBelowThreshold(t *testing.T) {
	score := &model.Score{Criteria: []model.CriterionScore{
		{CriterionKey: "pii", Weight: 2, MaxPoints: 5, Points: 5, Mandatory: true},
		{CriterionKey: "tone", Weight: 1, MaxPoints: 5, Points: 0},
		{CriterionKey: "closing", Weight: 1, MaxPoints: 5, Points: 0},
	}}

	score.Evaluate(0.9)

	if score.Passed {
		t.Fatal("eşiğin altındaki toplam geçmemeli")
	}
}

func TestScore_CountsUnverifiedQuotes(t *testing.T) {
	score := &model.Score{Criteria: []model.CriterionScore{
		{QuoteVerified: true}, {QuoteVerified: false}, {QuoteVerified: false},
	}}

	if got := score.UnverifiedQuoteCount(); got != 2 {
		t.Fatalf("doğrulanmamış alıntı sayısı 2 olmalı, %d geldi", got)
	}
}

// Hızlı kademe, karakter yanıtının yanında rubrik sinyallerini de döndürmeli.
func TestParseCharacterReply_ExtractsReplyAndSignals(t *testing.T) {
	raw := `{"reply":"Ama ben eşiyim!","signals":["pii","tone"]}`

	reply := ParseCharacterReply(raw, testRubric())

	if reply.Reply != "Ama ben eşiyim!" {
		t.Fatalf("replik çözümlenmeli: %q", reply.Reply)
	}
	if len(reply.Signals) != 2 {
		t.Fatalf("iki sinyal beklenir: %v", reply.Signals)
	}
}

// Uydurma bir sinyal, var olmayan bir kriterin tetiklendiğini söyler ve
// yönlendirme kuralları buna bakıyor: süzülmemiş bir liste maliyeti model
// hayal gücüne bağlar.
func TestParseCharacterReply_DropsUnknownSignalKeys(t *testing.T) {
	raw := `{"reply":"Merhaba","signals":["pii","uydurma_anahtar","tone","pii"]}`

	reply := ParseCharacterReply(raw, testRubric())

	if len(reply.Signals) != 2 {
		t.Fatalf("bilinmeyen ve yinelenen anahtarlar atılmalı: %v", reply.Signals)
	}
}

// Bozuk bir sinyal listesi yüzünden oturumu düşürmek, çalışanın görüşmesini
// bir biçim hatasına kurban etmek olurdu.
func TestParseCharacterReply_FallsBackToRawTextOnMalformedJSON(t *testing.T) {
	raw := "Ama ben eşiyim! Bu bilgiyi almam gerekiyor."

	reply := ParseCharacterReply(raw, testRubric())

	if reply.Reply != raw {
		t.Fatalf("JSON çözümlenemezse metnin tamamı replik olmalı: %q", reply.Reply)
	}
	if reply.Signals == nil {
		t.Fatal("sinyal listesi nil değil boş olmalı")
	}
}
