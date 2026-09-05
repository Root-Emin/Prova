package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// OverrideScoreUseCase, yöneticinin makine puanını ezmesi.
//
// Ezme puanı DEĞİŞTİRMEZ, üzerine yazar ve eskisini saklar. Makine sonucu
// görülemezse ezme yetkisi denetlenemez bir yetkiye dönüşür: bir yöneticinin
// herkesi geçirdiği bir kurulumla, modelin herkesi geçirdiği bir kurulum
// ayırt edilemez hâle gelir.
type OverrideScoreUseCase struct {
	scores   provaRepo.ScoreRepository
	sessions provaRepo.SessionRepository
	rubrics  provaRepo.RubricRepository
	audit    auditService.Recorder
	now      func() time.Time
}

// NewOverrideScoreUseCase wires the use case.
func NewOverrideScoreUseCase(
	scores provaRepo.ScoreRepository,
	sessions provaRepo.SessionRepository,
	rubrics provaRepo.RubricRepository,
	audit auditService.Recorder,
) *OverrideScoreUseCase {
	if audit == nil {
		audit = auditService.NoopRecorder{}
	}
	return &OverrideScoreUseCase{scores: scores, sessions: sessions, rubrics: rubrics, audit: audit, now: time.Now}
}

// OverrideRequest, ezme talebi.
type OverrideRequest struct {
	ScoreID uuid.UUID
	Total   float64
	Passed  bool
	Reason  string
}

// Execute, puanı ezer ve olayı denetime yazar.
func (uc *OverrideScoreUseCase) Execute(ctx context.Context, scope provaRepo.Scope, actorID uuid.UUID, req OverrideRequest) (*provaModel.Score, error) {
	// Gerekçe zorunlu ve boş bırakılamaz. Gerekçesiz bir ezme, denetim
	// kaydında "biri puanı değiştirdi" satırından fazlasını bırakmaz, ve o
	// satır bir sertifika itirazında hiçbir şey açıklamaz.
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, domainErr.New(domainErr.ErrValidation, "ezme gerekçesi zorunlu", nil)
	}

	existing, err := uc.scores.GetByID(ctx, scope, req.ScoreID)
	if err != nil {
		return nil, err
	}

	// Puan, rubriğin izin verdiği tavanı aşamaz. Aşabilseydi, ezilmiş bir
	// puan rubrikle karşılaştırılamaz hâle gelir ve kurum istatistiği
	// anlamını yitirirdi.
	if req.Total < 0 || req.Total > existing.MaxTotal {
		return nil, domainErr.New(domainErr.ErrValidation,
			"ezilen puan 0 ile rubriğin en yüksek puanı arasında olmalı", nil)
	}

	override := provaModel.ScoreOverride{
		OverriddenBy:  actorID,
		Reason:        reason,
		PreviousTotal: existing.Total,
		PreviousPass:  existing.Passed,
		OverriddenAt:  uc.now().UTC(),
	}

	updated, err := uc.scores.ApplyOverride(ctx, scope, req.ScoreID, req.Total, req.Passed, override)
	if err != nil {
		return nil, err
	}

	actor := actorID
	uc.audit.Record(ctx, auditService.Entry{
		OrgID:        scope.OrgID(),
		UserID:       &actor,
		Action:       auditService.ActionScoreOverridden,
		ResourceType: "score",
		ResourceID:   req.ScoreID.String(),
		Metadata: map[string]any{
			"session_id":     existing.SessionID.String(),
			"previous_total": existing.Total,
			"previous_pass":  existing.Passed,
			"new_total":      req.Total,
			"new_pass":       req.Passed,
			"reason":         reason,
		},
	})

	return updated, nil
}
