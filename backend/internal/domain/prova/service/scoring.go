package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// scoringSystemPrompt, puanlama kademesinin rolü.
//
// Karakter guardrail'i burada kullanılmıyor: puanlayıcı rol yapmıyor, bir
// değerlendirici. Ama kendi kısıtı var ve en önemlisi alıntı kuralı —
// alıntı, doğrulanabilir olduğu için istenmiyor; doğrulanabilir olduğu için
// puanı savunulabilir kılıyor.
const scoringSystemPrompt = `Sen bir kurumsal eğitim değerlendiricisisin.
Sana bir rol yapma görüşmesinin transkripti ve bir değerlendirme rubriği
verilecek. Görevin, çalışanın (EMPLOYEE) davranışını rubriğe göre puanlamak.

Kurallar:
1. Yalnızca transkriptte GEÇEN şeye göre puan ver. Olmayan bir davranışı
   varsaymazsın, olan bir davranışı yok saymazsın.
2. Her kriter için gerekçe olarak transkriptten BİREBİR bir alıntı verirsin
   ve o alıntının hangi konuşma sırasından geldiğini yazarsın. Alıntı
   transkriptte kelimesi kelimesine geçmelidir; özetleme, kısaltma ya da
   yeniden yazma yapmazsın.
3. Bir kriter için transkriptte hiçbir dayanak yoksa puanı düşük verirsin ve
   alıntı alanını boş bırakırsın. Uydurma alıntı vermek, o kriteri sıfır
   vermekten çok daha kötüdür.
4. Karakterin (CHARACTER) söylediklerini değil, çalışanın söylediklerini
   puanlarsın.
5. Yalnızca istenen JSON'u üretirsin; başka hiçbir metin eklemezsin.`

// ScoringInput, puanlama çağrısı için gereken her şey.
type ScoringInput struct {
	Scenario *model.Scenario
	Rubric   *model.Rubric
	// Turns, tam transkript (LLM'e giden maskelenmiş kopya).
	Turns []*model.Turn
	// AdminSuffix, LLM profilindeki yönetici eki.
	AdminSuffix string
}

// BuildScoringMessages, güçlü kademeye gönderilecek mesajları kurar.
func BuildScoringMessages(in ScoringInput) []Message {
	system := scoringSystemPrompt
	if strings.TrimSpace(in.AdminSuffix) != "" {
		system += "\n\nEK TALİMATLAR\n" + strings.TrimSpace(in.AdminSuffix)
	}

	var user strings.Builder
	user.WriteString("SENARYO\n")
	user.WriteString(in.Scenario.Title)
	user.WriteString("\n\nBAĞLAM\n")
	user.WriteString(in.Scenario.Context)
	if strings.TrimSpace(in.Scenario.Objective) != "" {
		user.WriteString("\n\nÇALIŞANDAN BEKLENEN\n")
		user.WriteString(in.Scenario.Objective)
	}

	user.WriteString("\n\nRUBRİK\n")
	for _, c := range in.Rubric.Criteria {
		fmt.Fprintf(&user, "- anahtar: %s\n  başlık: %s\n  açıklama: %s\n  en yüksek puan: %.0f%s\n",
			c.Key, c.Title, c.Description, c.MaxPoints, mandatoryNote(c.Mandatory))
	}

	user.WriteString("\nTRANSKRİPT\n")
	for _, turn := range in.Turns {
		if turn == nil {
			continue
		}
		speaker := "EMPLOYEE"
		if turn.Role == model.TurnRoleCharacter {
			speaker = "CHARACTER"
		}
		fmt.Fprintf(&user, "[%d] %s: %s\n", turn.Index, speaker, turn.Text)
	}

	user.WriteString("\n")
	user.WriteString(scoringResponseFormat)

	return []Message{
		{Role: RoleSystem, Content: system},
		{Role: RoleUser, Content: user.String()},
	}
}

func mandatoryNote(mandatory bool) string {
	if mandatory {
		return "\n  ZORUNLU: bu kriter düşerse sonuç doğrudan başarısızdır"
	}
	return ""
}

const scoringResponseFormat = `Yanıtını yalnızca şu JSON nesnesi olarak ver:
{
  "criteria": [
    {
      "key": "rubrikteki kriter anahtarı",
      "points": 0,
      "rationale": "puanın kısa gerekçesi",
      "quote": "transkriptten birebir alıntı (dayanak yoksa boş metin)",
      "turn_index": 0
    }
  ]
}
Rubrikteki her kriter için tam olarak bir nesne üret. "turn_index", alıntının
alındığı köşeli parantez içindeki sıra numarasıdır.`

// scoringPayload, modelden beklenen JSON gövdesi.
type scoringPayload struct {
	Criteria []struct {
		Key       string   `json:"key"`
		Points    float64  `json:"points"`
		Rationale string   `json:"rationale"`
		Quote     string   `json:"quote"`
		TurnIndex *float64 `json:"turn_index"`
	} `json:"criteria"`
}

// ParseScoring, modelin yanıtını kriter puanlarına çevirir.
//
// Rubrik referans alınıyor, modelin döndürdüğü liste değil: model bir kriteri
// atlarsa o kriter sessizce kaybolmamalı, sıfır puanla ve dayanaksız olarak
// görünmeli. Aksi hâlde bir kriteri atlayan model, o kriterden ceza almamış
// olurdu — ve zorunlu bir kriteri atlamak, KALDI kuralını atlamanın yolu
// hâline gelirdi.
func ParseScoring(raw string, rubric *model.Rubric) ([]model.CriterionScore, error) {
	var payload scoringPayload
	if err := json.Unmarshal([]byte(stripCodeFence(raw)), &payload); err != nil {
		return nil, fmt.Errorf("puanlama yanıtı çözümlenemedi: %w", err)
	}

	byKey := make(map[string]int, len(payload.Criteria))
	for i, item := range payload.Criteria {
		byKey[strings.TrimSpace(item.Key)] = i
	}

	scores := make([]model.CriterionScore, 0, len(rubric.Criteria))
	for _, criterion := range rubric.Criteria {
		score := model.CriterionScore{
			CriterionKey: criterion.Key,
			Title:        criterion.Title,
			Weight:       criterion.Weight,
			Mandatory:    criterion.Mandatory,
			MaxPoints:    criterion.MaxPoints,
		}

		index, found := byKey[criterion.Key]
		if !found {
			score.Rationale = "Model bu kriter için değerlendirme döndürmedi."
			scores = append(scores, score)
			continue
		}

		item := payload.Criteria[index]
		score.Points = clampPoints(item.Points, criterion.MaxPoints)
		score.Rationale = strings.TrimSpace(item.Rationale)
		score.Quote = strings.TrimSpace(item.Quote)
		if item.TurnIndex != nil {
			turn := int(*item.TurnIndex)
			score.TurnIndex = &turn
		}
		scores = append(scores, score)
	}

	return scores, nil
}

// clampPoints, modelin aralık dışına taşan puanını sınırlar.
//
// Bir modelin "10 üzerinden 12" vermesi nadir ama olur, ve o puan doğrudan
// toplama girseydi geçme eşiği anlamsızlaşırdı.
func clampPoints(points, max float64) float64 {
	switch {
	case points < 0:
		return 0
	case points > max:
		return max
	default:
		return points
	}
}
