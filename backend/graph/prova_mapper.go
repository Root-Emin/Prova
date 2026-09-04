package graph

import (
	"github.com/masterfabric-go/masterfabric/graph/model"
	provaUC "github.com/masterfabric-go/masterfabric/internal/application/prova/usecase"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

func mapDocumentStatus(s provaModel.DocumentStatus) model.DocumentStatus {
	switch s {
	case provaModel.DocumentStatusPublished:
		return model.DocumentStatusPublished
	case provaModel.DocumentStatusArchived:
		return model.DocumentStatusArchived
	default:
		return model.DocumentStatusDraft
	}
}

func mapDifficulty(d provaModel.Difficulty) model.Difficulty {
	switch d {
	case provaModel.DifficultyLow:
		return model.DifficultyLow
	case provaModel.DifficultyHigh:
		return model.DifficultyHigh
	case provaModel.DifficultyExtreme:
		return model.DifficultyExtreme
	default:
		return model.DifficultyMedium
	}
}

func toDifficulty(d model.Difficulty) provaModel.Difficulty {
	switch d {
	case model.DifficultyLow:
		return provaModel.DifficultyLow
	case model.DifficultyHigh:
		return provaModel.DifficultyHigh
	case model.DifficultyExtreme:
		return provaModel.DifficultyExtreme
	default:
		return provaModel.DifficultyMedium
	}
}

func mapTier(t provaModel.Tier) model.LLMTier {
	if t == provaModel.TierStrong {
		return model.LLMTierStrong
	}
	return model.LLMTierFast
}

func toTier(t model.LLMTier) provaModel.Tier {
	if t == model.LLMTierStrong {
		return provaModel.TierStrong
	}
	return provaModel.TierFast
}

func mapRef(r provaModel.Reference) *model.DocumentRef {
	return &model.DocumentRef{LineageID: r.LineageID, VersionID: r.VersionID, Version: r.Version}
}

func mapCharacter(c *provaModel.Character) *model.Character {
	if c == nil {
		return nil
	}
	return &model.Character{
		ID:            c.ID,
		LineageID:     c.LineageID,
		Version:       c.Version,
		Status:        mapDocumentStatus(c.Status),
		Name:          c.Name,
		Persona:       c.Persona,
		BehaviorRules: orEmpty(c.BehaviorRules),
		Difficulty:    mapDifficulty(c.Difficulty),
		HiddenFacts:   orEmpty(c.HiddenFacts),
		CreatedAt:     c.CreatedAt,
		PublishedAt:   c.PublishedAt,
	}
}

func mapCharacters(items []*provaModel.Character) []*model.Character {
	out := make([]*model.Character, 0, len(items))
	for _, item := range items {
		out = append(out, mapCharacter(item))
	}
	return out
}

func mapRubric(r *provaModel.Rubric) *model.Rubric {
	if r == nil {
		return nil
	}
	criteria := make([]*model.Criterion, 0, len(r.Criteria))
	for _, c := range r.Criteria {
		criterion := &model.Criterion{
			Key:         c.Key,
			Title:       c.Title,
			Description: c.Description,
			Weight:      c.Weight,
			MaxPoints:   c.MaxPoints,
			Mandatory:   c.Mandatory,
		}
		if c.Trap != "" {
			trap := c.Trap
			criterion.Trap = &trap
		}
		criteria = append(criteria, criterion)
	}
	return &model.Rubric{
		ID:            r.ID,
		LineageID:     r.LineageID,
		Version:       r.Version,
		Status:        mapDocumentStatus(r.Status),
		Name:          r.Name,
		Description:   r.Description,
		PassThreshold: r.PassThreshold,
		Criteria:      criteria,
		CreatedAt:     r.CreatedAt,
		PublishedAt:   r.PublishedAt,
	}
}

func mapRubrics(items []*provaModel.Rubric) []*model.Rubric {
	out := make([]*model.Rubric, 0, len(items))
	for _, item := range items {
		out = append(out, mapRubric(item))
	}
	return out
}

func mapScenario(s *provaModel.Scenario) *model.Scenario {
	if s == nil {
		return nil
	}
	return &model.Scenario{
		ID:           s.ID,
		LineageID:    s.LineageID,
		Version:      s.Version,
		Status:       mapDocumentStatus(s.Status),
		Title:        s.Title,
		Context:      s.Context,
		Objective:    s.Objective,
		CharacterRef: mapRef(s.CharacterRef),
		RubricRef:    mapRef(s.RubricRef),
		MaxTurns:     s.MaxTurns,
		CreatedAt:    s.CreatedAt,
		PublishedAt:  s.PublishedAt,
	}
}

func mapScenarios(items []*provaModel.Scenario) []*model.Scenario {
	out := make([]*model.Scenario, 0, len(items))
	for _, item := range items {
		out = append(out, mapScenario(item))
	}
	return out
}

func mapLLMProfile(p *provaModel.LLMProfile) *model.LLMProfile {
	if p == nil {
		return nil
	}
	return &model.LLMProfile{
		ID:                 p.ID,
		LineageID:          p.LineageID,
		Version:            p.Version,
		Status:             mapDocumentStatus(p.Status),
		Tier:               mapTier(p.Tier),
		Provider:           p.Provider,
		BaseURL:            p.BaseURL,
		Model:              p.Model,
		Temperature:        p.Temperature,
		TopP:               p.TopP,
		MaxTokens:          p.MaxTokens,
		SystemPromptSuffix: p.SystemPromptSuffix,
		InputCostPer1k:     p.InputCostPer1K,
		OutputCostPer1k:    p.OutputCostPer1K,
		CreatedAt:          p.CreatedAt,
	}
}

func mapLLMProfiles(items []*provaModel.LLMProfile) []*model.LLMProfile {
	out := make([]*model.LLMProfile, 0, len(items))
	for _, item := range items {
		out = append(out, mapLLMProfile(item))
	}
	return out
}

func mapTurn(t *provaModel.Turn) *model.Turn {
	if t == nil {
		return nil
	}
	role := model.TurnRoleEmployee
	if t.Role == provaModel.TurnRoleCharacter {
		role = model.TurnRoleCharacter
	}
	return &model.Turn{
		Index:            t.Index,
		Role:             role,
		Text:             t.Text,
		Signals:          orEmpty(t.Signals),
		MaskedFieldCount: t.MaskedFieldCount,
		CreatedAt:        t.CreatedAt,
	}
}

func mapTurns(items []*provaModel.Turn) []*model.Turn {
	out := make([]*model.Turn, 0, len(items))
	for _, item := range items {
		out = append(out, mapTurn(item))
	}
	return out
}

func mapScore(s *provaModel.Score) *model.Score {
	if s == nil {
		return nil
	}
	criteria := make([]*model.CriterionScore, 0, len(s.Criteria))
	for _, c := range s.Criteria {
		criteria = append(criteria, &model.CriterionScore{
			CriterionKey:  c.CriterionKey,
			Title:         c.Title,
			Weight:        c.Weight,
			Mandatory:     c.Mandatory,
			Points:        c.Points,
			MaxPoints:     c.MaxPoints,
			Rationale:     c.Rationale,
			Quote:         c.Quote,
			TurnIndex:     c.TurnIndex,
			QuoteVerified: c.QuoteVerified,
		})
	}
	out := &model.Score{
		ID:                  s.ID,
		SessionID:           s.SessionID,
		Total:               s.Total,
		MaxTotal:            s.MaxTotal,
		Passed:              s.Passed,
		FailedMandatoryKeys: orEmpty(s.FailedMandatoryKeys),
		Criteria:            criteria,
		Model:               s.Model,
		CreatedAt:           s.CreatedAt,
	}
	if s.Override != nil {
		out.Override = &model.ScoreOverride{
			OverriddenBy:   s.Override.OverriddenBy,
			Reason:         s.Override.Reason,
			PreviousTotal:  s.Override.PreviousTotal,
			PreviousPassed: s.Override.PreviousPass,
			OverriddenAt:   s.Override.OverriddenAt,
		}
	}
	return out
}

func mapSessionStatus(s provaModel.SessionStatus) model.SessionStatus {
	switch s {
	case provaModel.SessionStatusScoring:
		return model.SessionStatusScoring
	case provaModel.SessionStatusCompleted:
		return model.SessionStatusCompleted
	case provaModel.SessionStatusAbandoned:
		return model.SessionStatusAbandoned
	default:
		return model.SessionStatusActive
	}
}

// mapSession, oturumu şema tipine çevirir. Konuşma sıraları ve puan ayrı
// resolver'lardan geliyor: liste sorgusunda her oturum için transkript
// çekmek, yirmi satırlık bir sayfayı yüzlerce belge okumaya çevirirdi.
func mapSession(s *provaModel.Session) *model.Session {
	if s == nil {
		return nil
	}
	return &model.Session{
		ID:         s.ID,
		Status:     mapSessionStatus(s.Status),
		EmployeeID: s.EmployeeID,
		Scenario:   mapRef(s.Scenario),
		Rubric:     mapRef(s.Rubric),
		Character:  mapRef(s.Character),
		Turns:      []*model.Turn{},
		StartedAt:  s.StartedAt,
		EndedAt:    s.EndedAt,
	}
}

func mapSessions(items []*provaModel.Session) []*model.Session {
	out := make([]*model.Session, 0, len(items))
	for _, item := range items {
		out = append(out, mapSession(item))
	}
	return out
}

func mapSessionEvent(e provaUC.SessionEvent) *model.SessionEvent {
	out := &model.SessionEvent{
		SessionID:  e.SessionID,
		Type:       model.SessionEventType(e.Type),
		OccurredAt: e.OccurredAt,
		Turn:       mapTurn(e.Turn),
		Score:      mapScore(e.Score),
	}
	if e.Text != "" {
		text := e.Text
		out.Text = &text
	}
	if len(e.Signals) > 0 {
		out.Signals = e.Signals
	}
	if e.Progress > 0 {
		progress := e.Progress
		out.Progress = &progress
	}
	return out
}

// orEmpty, nil dilimi boş dilime çevirir.
//
// GraphQL'de nullable olmayan bir liste alanı nil dönemez; boş dilim
// döndürmek, her resolver'da ayrı bir nil kontrolü yazmaktan iyidir.
func orEmpty(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}
