package prova

import (
	"context"
	"errors"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
	provaMongo "github.com/masterfabric-go/masterfabric/internal/infrastructure/mongo"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CharacterRepo, karakter belgeleri.
type CharacterRepo struct {
	*VersionedRepo[model.Character, *model.Character]
}

// NewCharacterRepo wires the repository.
func NewCharacterRepo(db *mongo.Database) *CharacterRepo {
	return &CharacterRepo{NewVersionedRepo[model.Character, *model.Character](db, provaMongo.CollectionCharacters)}
}

// ScenarioRepo, senaryo belgeleri.
type ScenarioRepo struct {
	*VersionedRepo[model.Scenario, *model.Scenario]
}

// NewScenarioRepo wires the repository.
func NewScenarioRepo(db *mongo.Database) *ScenarioRepo {
	return &ScenarioRepo{NewVersionedRepo[model.Scenario, *model.Scenario](db, provaMongo.CollectionScenarios)}
}

// RubricRepo, rubrik belgeleri.
type RubricRepo struct {
	*VersionedRepo[model.Rubric, *model.Rubric]
}

// NewRubricRepo wires the repository.
func NewRubricRepo(db *mongo.Database) *RubricRepo {
	return &RubricRepo{NewVersionedRepo[model.Rubric, *model.Rubric](db, provaMongo.CollectionRubrics)}
}

// LLMProfileRepo, LLM profilleri.
type LLMProfileRepo struct {
	*VersionedRepo[model.LLMProfile, *model.LLMProfile]
}

// NewLLMProfileRepo wires the repository.
func NewLLMProfileRepo(db *mongo.Database) *LLMProfileRepo {
	return &LLMProfileRepo{NewVersionedRepo[model.LLMProfile, *model.LLMProfile](db, provaMongo.CollectionLLMProfiles)}
}

// GetPublishedByTier, bir kademenin yayınlanmış profilini döndürür.
//
// Aynı kademede birden fazla yayınlanmış profil varsa en yeni oluşturulan
// kazanır. Yönetici yeni bir profil yayınladığında eskisini ayrıca
// arşivlemek zorunda kalmamalı; yayınlamak yürürlüğe koymak demek.
func (r *LLMProfileRepo) GetPublishedByTier(ctx context.Context, scope repository.Scope, tier model.Tier) (*model.LLMProfile, error) {
	condition, err := r.filter(scope, bson.M{
		"tier":   tier,
		"status": model.DocumentStatusPublished,
	})
	if err != nil {
		return nil, err
	}

	var profile model.LLMProfile
	err = r.coll.FindOne(ctx, condition,
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "version", Value: -1}}),
	).Decode(&profile)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainErr.New(domainErr.ErrNotFound,
				string(tier)+" kademesi için yayınlanmış bir LLM profili yok", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "LLM profili okunamadı", err)
	}
	return &profile, nil
}

// Arayüz uyumu derleme zamanında kontrol ediliyor: bir metot imzası
// kaydığında hata, wiring sırasında değil burada çıksın.
var (
	_ repository.CharacterRepository  = (*CharacterRepo)(nil)
	_ repository.ScenarioRepository   = (*ScenarioRepo)(nil)
	_ repository.RubricRepository     = (*RubricRepo)(nil)
	_ repository.LLMProfileRepository = (*LLMProfileRepo)(nil)
)
