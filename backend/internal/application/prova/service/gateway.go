// Package service, LLM çağrılarını tek bir kapıdan geçiren geçidi barındırır.
//
// Her LLM çağrısı buradan geçer. Sebep tek: profil çözümü, PII maskeleme,
// kademe yönlendirmesi ve yönlendirme kaydı çağrı başına yapılmak zorunda, ve
// bunlardan birini unutan bir çağrı yolu sessizce yanlış davranır — yanlış
// modeli kullanır, maskelenmemiş kişisel veri gönderir ya da istatistikte hiç
// görünmez.
package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	provaService "github.com/masterfabric-go/masterfabric/internal/domain/prova/service"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// Purpose, çağrının ne için yapıldığı. Yönlendirme kuralları buna bakar.
type Purpose string

const (
	// PurposeTurn, konuşma sırasında karakteri canlandırma.
	PurposeTurn Purpose = "turn"
	// PurposeScoring, oturum sonu puanlama.
	PurposeScoring Purpose = "scoring"
	// PurposeProbe, yöneticinin profili canlıya almadan denemesi.
	PurposeProbe Purpose = "probe"
)

// RouteInput, yönlendirme kararının girdisi.
type RouteInput struct {
	Purpose Purpose
	// MandatorySignal, bu sırada zorunlu bir kriterin tetiklendiği.
	MandatorySignal bool
	// InputLength, gönderilecek metnin karakter sayısı.
	InputLength int
}

// Router, çağrıyı bir kademeye yönlendirir.
//
// Port olarak duruyor çünkü kural motoru ve devre kesici ayrı bir fazın işi;
// geçit yalnızca "hangi kademe ve neden" cevabına ihtiyaç duyuyor.
type Router interface {
	// Route, kademeyi ve kararı veren kuralı döndürür.
	Route(ctx context.Context, in RouteInput) (provaModel.Tier, provaModel.RoutingRule)
	// ReportResult, devre kesicinin sayaçlarını besler.
	ReportResult(tier provaModel.Tier, success bool)
	// IsOpen, kademenin devresinin açık (kullanılamaz) olduğunu söyler.
	IsOpen(tier provaModel.Tier) bool
}

// Masker, LLM'e giden metinden kişisel veriyi temizler.
//
// Yalnızca giden kopyada çalışır; transkriptin kendisi maskelenmez, çünkü
// "çalışan kişisel veri ifşa etti mi" sorusu ancak orijinal metinle
// cevaplanabilir.
type Masker interface {
	// Mask, metni maskeler ve kaç alanın maskelendiğini döndürür.
	Mask(text string) (masked string, fields int)
}

// APIKeys, sağlayıcı kimlik bilgilerini çözer.
//
// Anahtarlar profilde değil env'de: bir sır veritabanına yazılırsa
// veritabanı yedeği sırların yedeği hâline gelir.
type APIKeys interface {
	For(provider string) string
}

// Call, geçitten geçen tek bir çağrı.
type Call struct {
	Scope     provaRepo.Scope
	SessionID *uuid.UUID
	Purpose   Purpose
	Messages  []provaService.Message
	// MandatorySignal, yönlendirme kuralına girdi.
	MandatorySignal bool
	// Difficulty, üretim sıcaklığını ayarlar. Puanlamada boş bırakılır.
	Difficulty provaModel.Difficulty
	JSONMode   bool
	// ForceProfile, yöneticinin test ettiği profil. Doluysa yönlendirme
	// atlanır: test edilen şey tam olarak o profildir.
	ForceProfile *provaModel.LLMProfile
	// OnDelta doluysa akışlı çağrı yapılır.
	OnDelta func(string) error
}

// Result, geçidin döndürdüğü sonuç.
type Result struct {
	Content string
	Tier    provaModel.Tier
	Rule    provaModel.RoutingRule
	Model   string
	Profile *provaModel.LLMProfile
	Usage   provaService.Usage
	Latency time.Duration
	CostUSD float64
	// MaskedFields, bu çağrıda maskelenen kişisel veri alanı sayısı.
	MaskedFields int
}

// Gateway, LLM çağrılarının tek kapısı.
type Gateway struct {
	profiles provaRepo.LLMProfileRepository
	routing  provaRepo.RoutingRepository
	provider provaService.Provider
	router   Router
	masker   Masker
	keys     APIKeys
	log      *slog.Logger
	now      func() time.Time
}

// NewGateway wires the gateway.
func NewGateway(
	profiles provaRepo.LLMProfileRepository,
	routing provaRepo.RoutingRepository,
	provider provaService.Provider,
	router Router,
	masker Masker,
	keys APIKeys,
	log *slog.Logger,
) *Gateway {
	return &Gateway{
		profiles: profiles,
		routing:  routing,
		provider: provider,
		router:   router,
		masker:   masker,
		keys:     keys,
		log:      log,
		now:      time.Now,
	}
}

// Invoke, çağrıyı yönlendirir, çalıştırır ve kaydeder.
//
// Hızlı kademe başarısız olursa güçlü kademeye düşülür ve bu, failover olarak
// kaydedilir. Oturumun ortasında bir sağlayıcı hatası yüzünden görüşmeyi
// kesmek, çalışanın o ana kadarki emeğini çöpe atmak olurdu.
func (g *Gateway) Invoke(ctx context.Context, call Call) (*Result, error) {
	masked, maskedFields := g.maskMessages(call.Messages)

	tier, rule := g.decide(ctx, call, masked)

	result, err := g.attempt(ctx, call, masked, maskedFields, tier, rule)
	if err == nil {
		return result, nil
	}

	// Failover yalnızca hızlıdan güçlüye. Güçlü kademe düşerse düşülecek bir
	// yer yok, ve puanlamayı hızlı modele yaptırmak sertifikayı değersiz
	// kılardı.
	if tier != provaModel.TierFast {
		return nil, err
	}
	g.log.WarnContext(ctx, "hızlı kademe başarısız, güçlü kademeye düşülüyor", "error", err)

	return g.attempt(ctx, call, masked, maskedFields, provaModel.TierStrong, provaModel.RuleFailover)
}

// decide, kademeyi ve kuralı belirler.
func (g *Gateway) decide(ctx context.Context, call Call, masked []provaService.Message) (provaModel.Tier, provaModel.RoutingRule) {
	if call.ForceProfile != nil {
		return call.ForceProfile.Tier, provaModel.RuleDefault
	}
	return g.router.Route(ctx, RouteInput{
		Purpose:         call.Purpose,
		MandatorySignal: call.MandatorySignal,
		InputLength:     totalLength(masked),
	})
}

// attempt, tek bir kademeye çağrı yapar ve sonucu kaydeder.
func (g *Gateway) attempt(
	ctx context.Context,
	call Call,
	messages []provaService.Message,
	maskedFields int,
	tier provaModel.Tier,
	rule provaModel.RoutingRule,
) (*Result, error) {
	profile := call.ForceProfile
	if profile == nil {
		resolved, err := g.profiles.GetPublishedByTier(ctx, call.Scope, tier)
		if err != nil {
			return nil, err
		}
		profile = resolved
	}

	req := provaService.CompletionRequest{
		Target: provaService.Target{
			BaseURL: profile.BaseURL,
			APIKey:  g.keys.For(profile.Provider),
			Model:   profile.Model,
		},
		Messages:    messages,
		Temperature: profile.TemperatureFor(call.Difficulty),
		TopP:        profile.TopP,
		MaxTokens:   profile.MaxTokens,
		JSONMode:    call.JSONMode,
	}

	started := g.now()
	var (
		response *provaService.CompletionResponse
		err      error
	)
	if call.OnDelta != nil {
		response, err = g.provider.Stream(ctx, req, call.OnDelta)
	} else {
		response, err = g.provider.Complete(ctx, req)
	}

	g.router.ReportResult(tier, err == nil)

	if err != nil {
		g.record(ctx, call, profile, tier, rule, provaService.Usage{}, g.now().Sub(started), maskedFields, err)
		return nil, err
	}

	cost := profile.Cost(response.Usage.InputTokens, response.Usage.OutputTokens)
	g.record(ctx, call, profile, tier, rule, response.Usage, response.Latency, maskedFields, nil)

	return &Result{
		Content:      response.Content,
		Tier:         tier,
		Rule:         rule,
		Model:        response.Model,
		Profile:      profile,
		Usage:        response.Usage,
		Latency:      response.Latency,
		CostUSD:      cost,
		MaskedFields: maskedFields,
	}, nil
}

// record, yönlendirme kararını kaydeder.
//
// Başarısız çağrılar da kaydediliyor: failover'ın gerçekten olduğunu
// gösteren tek kanıt, düşen çağrının kaydı.
func (g *Gateway) record(
	ctx context.Context,
	call Call,
	profile *provaModel.LLMProfile,
	tier provaModel.Tier,
	rule provaModel.RoutingRule,
	usage provaService.Usage,
	latency time.Duration,
	maskedFields int,
	callErr error,
) {
	if g.routing == nil {
		return
	}

	record := &provaModel.RoutingRecord{
		SessionID:        call.SessionID,
		Tier:             tier,
		Rule:             rule,
		Model:            profile.Model,
		ProfileID:        profile.ID,
		LatencyMs:        int(latency.Milliseconds()),
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		CostUSD:          profile.Cost(usage.InputTokens, usage.OutputTokens),
		Success:          callErr == nil,
		MaskedFieldCount: maskedFields,
	}
	if callErr != nil {
		record.Error = callErr.Error()
	}

	// Kayıt, isteğin bağlamından bağımsız yazılıyor: istemci bağlantıyı
	// kesse bile karar verilmiş ve para harcanmıştır.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), recordTimeout)
	defer cancel()

	if err := g.routing.Record(writeCtx, call.Scope, record); err != nil {
		g.log.ErrorContext(writeCtx, "yönlendirme kaydı yazılamadı", "tier", tier, "error", err)
	}
}

// recordTimeout, yönlendirme kaydının yazımı için üst sınır.
const recordTimeout = 5 * time.Second

// maskMessages, kullanıcıdan gelen mesajları maskeler.
//
// Sistem mesajı maskelenmiyor: içeriği bizim yazdığımız prompt, ve
// maskeleyici oradaki örnek biçimleri (rubrikteki tuzak metni gibi) bozabilir.
// Kişisel veri kullanıcı mesajlarından gelir.
func (g *Gateway) maskMessages(messages []provaService.Message) ([]provaService.Message, int) {
	if g.masker == nil {
		return messages, 0
	}

	out := make([]provaService.Message, len(messages))
	var total int
	for i, m := range messages {
		if m.Role == provaService.RoleSystem {
			out[i] = m
			continue
		}
		maskedText, fields := g.masker.Mask(m.Content)
		out[i] = provaService.Message{Role: m.Role, Content: maskedText}
		total += fields
	}
	return out, total
}

func totalLength(messages []provaService.Message) int {
	var total int
	for _, m := range messages {
		total += len(m.Content)
	}
	return total
}

// ErrNoProvider, sağlayıcı bağlı değilken döndürülür.
var ErrNoProvider = domainErr.New(domainErr.ErrNotImplemented,
	"LLM sağlayıcısı yapılandırılmamış", nil)
