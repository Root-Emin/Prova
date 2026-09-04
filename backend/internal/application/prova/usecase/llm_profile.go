package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	appService "github.com/masterfabric-go/masterfabric/internal/application/prova/service"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	provaService "github.com/masterfabric-go/masterfabric/internal/domain/prova/service"
)

// LLMProfileUseCase, model ayarlarının çalışma anındaki yönetimi.
//
// Model kimliği, sağlayıcı, sıcaklık, topP, maksimum token ve sistem prompt
// eki buradan değiştirilir. Yeniden derleme ya da yeniden başlatma gerekmez:
// değerler her çağrıda profilden okunuyor.
type LLMProfileUseCase struct {
	content *ContentUseCase[provaModel.LLMProfile, *provaModel.LLMProfile]
	repo    provaRepo.LLMProfileRepository
	gateway *appService.Gateway
	audit   auditService.Recorder
	now     func() time.Time
}

// NewLLMProfileUseCase wires the use case.
func NewLLMProfileUseCase(
	repo provaRepo.LLMProfileRepository,
	gateway *appService.Gateway,
	audit auditService.Recorder,
) *LLMProfileUseCase {
	if audit == nil {
		audit = auditService.NoopRecorder{}
	}
	return &LLMProfileUseCase{
		content: NewContentUseCase[provaModel.LLMProfile](repo, audit, "llm_profile"),
		repo:    repo,
		gateway: gateway,
		audit:   audit,
		now:     time.Now,
	}
}

// ProfileSettings, yöneticinin değiştirebildiği alanlar.
type ProfileSettings struct {
	Tier               provaModel.Tier
	Provider           string
	BaseURL            string
	Model              string
	Temperature        float64
	TopP               float64
	MaxTokens          int
	SystemPromptSuffix string
	InputCostPer1K     float64
	OutputCostPer1K    float64
}

func (s ProfileSettings) applyTo(p *provaModel.LLMProfile) {
	p.Tier = s.Tier
	p.Provider = s.Provider
	p.BaseURL = s.BaseURL
	p.Model = s.Model
	p.Temperature = s.Temperature
	p.TopP = s.TopP
	p.MaxTokens = s.MaxTokens
	p.SystemPromptSuffix = s.SystemPromptSuffix
	p.InputCostPer1K = s.InputCostPer1K
	p.OutputCostPer1K = s.OutputCostPer1K
}

// Create, yeni bir profil soyu açar.
func (uc *LLMProfileUseCase) Create(ctx context.Context, scope provaRepo.Scope, userID uuid.UUID, settings ProfileSettings) (*provaModel.LLMProfile, error) {
	profile, err := uc.content.Create(ctx, scope, userID, settings.applyTo)
	if err != nil {
		return nil, err
	}
	uc.recordProfileChange(ctx, scope, userID, profile, settings, "created")
	return profile, nil
}

// Update, profili düzenler ve yeni sürüm üretir.
func (uc *LLMProfileUseCase) Update(ctx context.Context, scope provaRepo.Scope, userID, lineageID uuid.UUID, settings ProfileSettings) (*provaModel.LLMProfile, error) {
	profile, err := uc.content.Update(ctx, scope, userID, lineageID, settings.applyTo)
	if err != nil {
		return nil, err
	}
	uc.recordProfileChange(ctx, scope, userID, profile, settings, "updated")
	return profile, nil
}

// Publish, profili yürürlüğe koyar.
func (uc *LLMProfileUseCase) Publish(ctx context.Context, scope provaRepo.Scope, userID, lineageID uuid.UUID, version int) (*provaModel.LLMProfile, error) {
	return uc.content.Publish(ctx, scope, userID, lineageID, version)
}

// ProbeResult, tek seferlik deneme sonucu.
type ProbeResult struct {
	ProfileID    uuid.UUID
	Model        string
	Output       string
	LatencyMs    int
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Error        string
}

// Probe, profili canlıya almadan tek seferlik dener.
//
// Yayınlanmamış bir sürüm de denenebilir; denemenin bütün amacı bu. Çağrı
// yönlendirmeyi atlar ve doğrudan verilen profile gider, çünkü test edilen
// şey tam olarak o profil.
func (uc *LLMProfileUseCase) Probe(ctx context.Context, scope provaRepo.Scope, userID, lineageID uuid.UUID, prompt string) (*ProbeResult, error) {
	profile, err := uc.repo.GetLatest(ctx, scope, lineageID)
	if err != nil {
		return nil, err
	}

	messages := []provaService.Message{
		{
			Role: provaService.RoleSystem,
			Content: "Bu bir profil testidir. Kısa ve doğrudan yanıt ver." +
				suffixSection(profile.SystemPromptSuffix),
		},
		{Role: provaService.RoleUser, Content: prompt},
	}

	result := &ProbeResult{ProfileID: profile.ID, Model: profile.Model}

	started := uc.now()
	invoked, err := uc.gateway.Invoke(ctx, appService.Call{
		Scope:        scope,
		Purpose:      appService.PurposeProbe,
		Messages:     messages,
		ForceProfile: profile,
	})
	if err != nil {
		// Deneme hatası bir uygulama hatası değil, denemenin sonucu:
		// yönetici yanlış bir base URL girdiyse bunu hata mesajı olarak
		// görmeli, boş bir ekran olarak değil.
		result.Error = err.Error()
		result.LatencyMs = int(uc.now().Sub(started).Milliseconds())
		return result, nil
	}

	result.Output = invoked.Content
	result.LatencyMs = int(invoked.Latency.Milliseconds())
	result.InputTokens = invoked.Usage.InputTokens
	result.OutputTokens = invoked.Usage.OutputTokens
	result.CostUSD = invoked.CostUSD

	actor := userID
	uc.audit.Record(ctx, auditService.Entry{
		OrgID:        scope.OrgID(),
		UserID:       &actor,
		Action:       auditService.ActionLLMProfileTested,
		ResourceType: "llm_profile",
		ResourceID:   profile.LineageID.String(),
		Metadata: map[string]any{
			"version":    profile.Version,
			"model":      profile.Model,
			"latency_ms": result.LatencyMs,
		},
	})

	return result, nil
}

// recordProfileChange, denetim kaydına neyin değiştiğini yazar.
//
// Prompt eki metninin kendisi kaydedilmiyor, yalnızca uzunluğu: denetim
// kaydı silinmeyen tek koleksiyon ve içine yazılan her serbest metin kalıcı.
// Neyin değiştiğini görmek için sürümlü belgeye bakılır.
func (uc *LLMProfileUseCase) recordProfileChange(
	ctx context.Context,
	scope provaRepo.Scope,
	userID uuid.UUID,
	profile *provaModel.LLMProfile,
	settings ProfileSettings,
	change string,
) {
	actor := userID
	uc.audit.Record(ctx, auditService.Entry{
		OrgID:        scope.OrgID(),
		UserID:       &actor,
		Action:       auditService.ActionLLMProfileChanged,
		ResourceType: "llm_profile",
		ResourceID:   profile.LineageID.String(),
		Metadata: map[string]any{
			"change":               change,
			"version":              profile.Version,
			"tier":                 string(settings.Tier),
			"provider":             settings.Provider,
			"model":                settings.Model,
			"temperature":          settings.Temperature,
			"top_p":                settings.TopP,
			"max_tokens":           settings.MaxTokens,
			"prompt_suffix_length": len(settings.SystemPromptSuffix),
		},
	})
}

func suffixSection(suffix string) string {
	if suffix == "" {
		return ""
	}
	return "\n\nEK TALİMATLAR\n" + suffix
}
