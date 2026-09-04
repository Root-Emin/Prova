package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// characterResponseFormat, hızlı kademeden istenen JSON biçiminin tarifi.
//
// Karakter yanıtının yanında tetiklenen rubrik sinyalleri de isteniyor.
// Sinyalleri ayrı bir çağrıyla sormak, her konuşma sırasını iki LLM çağrısına
// çıkarırdı; aynı çağrıda istemek hem maliyeti yarıya indirir hem de sinyali
// üreten modelin yanıtı üreten modelle aynı bağlamı görmesini sağlar.
const characterResponseFormat = `Yanıtını yalnızca şu JSON nesnesi olarak ver, başka hiçbir metin ekleme:
{
  "reply": "karakterin söyledikleri (yalnızca replik, tırnak ya da isim etiketi olmadan)",
  "signals": ["tetiklenen kriter anahtarları"]
}
"signals" alanına, çalışanın SON mesajında gözlemlediğin davranışların
kriter anahtarlarını yazarsın. Gözlem yoksa boş dizi verirsin. Anahtarları
uydurma; yalnızca aşağıda listelenenleri kullan.`

// CharacterTurnInput, bir konuşma sırası için gereken her şey.
type CharacterTurnInput struct {
	Character *model.Character
	Scenario  *model.Scenario
	Rubric    *model.Rubric
	// History, o ana kadarki transkript. LLM'e giden kopyada maskelenmiş
	// olabilir; transkriptin kendisi maskelenmez.
	History []*model.Turn
	// EmployeeMessage, çalışanın bu sıradaki mesajı (maskelenmiş kopya).
	EmployeeMessage string
	// AdminSuffix, LLM profilindeki yönetici eki.
	AdminSuffix string
}

// BuildCharacterMessages, hızlı kademeye gönderilecek mesaj dizisini kurar.
func BuildCharacterMessages(in CharacterTurnInput) []Message {
	systemPrompt := CompileCharacterPrompt(PromptParts{
		CharacterPersona:      in.Character.Persona,
		BehaviorRules:         in.Character.BehaviorRules,
		DifficultyInstruction: in.Character.Difficulty.BehaviourInstruction(),
		HiddenFacts:           in.Character.HiddenFacts,
		ScenarioContext:       in.Scenario.Context,
		RubricTraps:           in.Rubric.Traps(),
		AdminSuffix:           in.AdminSuffix,
		ResponseFormat:        characterResponseFormat + "\n\n" + criterionKeyList(in.Rubric),
	})

	messages := make([]Message, 0, len(in.History)+2)
	messages = append(messages, Message{Role: RoleSystem, Content: systemPrompt})

	for _, turn := range in.History {
		if turn == nil {
			continue
		}
		role := RoleUser
		if turn.Role == model.TurnRoleCharacter {
			role = RoleAssistant
		}
		messages = append(messages, Message{Role: role, Content: turn.Text})
	}

	messages = append(messages, Message{Role: RoleUser, Content: in.EmployeeMessage})
	return messages
}

// criterionKeyList, modelin kullanabileceği kriter anahtarlarını listeler.
func criterionKeyList(rubric *model.Rubric) string {
	var b strings.Builder
	b.WriteString("Kullanabileceğin kriter anahtarları:")
	for _, c := range rubric.Criteria {
		fmt.Fprintf(&b, "\n- %s: %s", c.Key, c.Title)
	}
	return b.String()
}

// CharacterReply, hızlı kademenin çözümlenmiş yanıtı.
type CharacterReply struct {
	Reply   string
	Signals []string
}

// characterReplyPayload, modelden beklenen JSON gövdesi.
type characterReplyPayload struct {
	Reply   string   `json:"reply"`
	Signals []string `json:"signals"`
}

// ParseCharacterReply, modelin JSON yanıtını çözümler.
//
// Model JSON modda çağrılıyor ama yine de savunmalı okunuyor: bazı
// sağlayıcılar JSON'u kod bloğuna sarıyor, bazıları biçim talimatını
// gözden kaçırıyor. Çözümleme başarısız olursa yanıtın tamamı replik olarak
// kabul ediliyor — bozuk bir sinyal listesi yüzünden oturumu düşürmek,
// çalışanın görüşmesini bir biçim hatasına kurban etmek olurdu.
func ParseCharacterReply(raw string, rubric *model.Rubric) CharacterReply {
	cleaned := stripCodeFence(raw)

	var payload characterReplyPayload
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil || strings.TrimSpace(payload.Reply) == "" {
		return CharacterReply{Reply: strings.TrimSpace(raw), Signals: []string{}}
	}

	return CharacterReply{
		Reply:   strings.TrimSpace(payload.Reply),
		Signals: filterKnownKeys(payload.Signals, rubric),
	}
}

// filterKnownKeys, modelin uydurduğu anahtarları atar.
//
// Uydurma bir sinyal, var olmayan bir kriterin tetiklendiğini söyler ve
// yönlendirme kuralları (zorunlu kriter sinyali → güçlü kademe) buna bakıyor.
// Süzülmemiş bir liste, maliyeti model hayal gücüne bağlar.
func filterKnownKeys(signals []string, rubric *model.Rubric) []string {
	out := make([]string, 0, len(signals))
	seen := make(map[string]bool, len(signals))
	for _, key := range signals {
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			continue
		}
		if _, ok := rubric.CriterionByKey(key); !ok {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

// stripCodeFence, JSON'u saran markdown kod bloğunu kaldırır.
func stripCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	trimmed = strings.TrimPrefix(trimmed, "```")
	if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
		// İlk satır dil etiketi olabilir ("json"); atılır.
		if label := strings.TrimSpace(trimmed[:idx]); label == "" || !strings.ContainsAny(label, "{[") {
			trimmed = trimmed[idx+1:]
		}
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(trimmed), "```"))
}
