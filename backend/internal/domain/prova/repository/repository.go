// Package repository, Prova belgelerinin kalıcılık portlarını tanımlar.
//
// Arayüzler domain katmanında duruyor; uygulama katmanı yalnızca bunları
// görür ve MongoDB'yi hiç bilmez. Bu, "nesne veritabanı" seçiminin bir
// uygulama detayı olarak kalmasını sağlar — ve testlerin veritabanı olmadan
// çalışabilmesini.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// Scope, her sorgunun taşımak zorunda olduğu kiracı sınırı.
//
// Ayrı bir tip olarak geçiriliyor, uuid.UUID olarak değil: bir metot imzasında
// iki uuid.UUID yan yana durduğunda hangisinin organizasyon olduğu yalnızca
// parametre adından anlaşılır, ve yanlış sırada geçirmek derlenir. Bu tip
// yanlış sırayı derlenemez yapar.
type Scope struct {
	orgID uuid.UUID
}

// NewScope binds queries to an organisation.
func NewScope(orgID uuid.UUID) Scope { return Scope{orgID: orgID} }

// OrgID returns the bound organisation.
func (s Scope) OrgID() uuid.UUID { return s.orgID }

// IsZero reports whether the scope names no organisation. A repository that
// receives one must refuse the query rather than run it unscoped.
func (s Scope) IsZero() bool { return s.orgID == uuid.Nil }

// ListOptions, listeleme sorgularının filtreleri.
type ListOptions struct {
	// PublishedOnly, yalnızca oynanabilir sürümleri döndürür.
	PublishedOnly bool
	// Limit ve Offset sayfalama içindir. Limit sıfırsa depo kendi
	// varsayılanını uygular; sınırsız liste döndüren bir sorgu yoktur.
	Limit  int
	Offset int
}

// Versioned, sürümlenebilir belgeler için ortak depo sözleşmesi.
//
// Generic: karakter, senaryo, rubrik ve LLM profili aynı sürümleme kurallarına
// uyar, ve o kuralları dört kez yazmak dördünün zamanla ayrışması demektir.
type Versioned[T any] interface {
	// Create, yeni bir soyun ilk taslağını yazar.
	Create(ctx context.Context, scope Scope, doc *T) error

	// UpdateDraft, taslağı yerinde günceller.
	//
	// Yayınlanmış bir belgeye çağrılırsa model.ErrDocumentFrozen döner.
	// Kural depo seviyesinde zorlanıyor, use case seviyesinde değil: tek bir
	// unutulmuş kontrol, sertifikanın dayandığı metnin sonradan yeniden
	// yazılabilmesi demek.
	UpdateDraft(ctx context.Context, scope Scope, doc *T) error

	// NewVersion, mevcut son sürümden yeni bir taslak türetir.
	NewVersion(ctx context.Context, scope Scope, lineageID, createdBy uuid.UUID, apply func(*T)) (*T, error)

	// GetLatest, soyun en yüksek sürüm numaralı belgesini döndürür.
	GetLatest(ctx context.Context, scope Scope, lineageID uuid.UUID) (*T, error)

	// GetLatestPublished, soyun oynanabilir son sürümünü döndürür.
	GetLatestPublished(ctx context.Context, scope Scope, lineageID uuid.UUID) (*T, error)

	// GetVersion, belirli bir sürümü döndürür.
	GetVersion(ctx context.Context, scope Scope, lineageID uuid.UUID, version int) (*T, error)

	// GetByID, sürüm kimliğiyle tek belge döndürür. Oturumun dondurduğu
	// referansı çözmek için kullanılır.
	GetByID(ctx context.Context, scope Scope, versionID uuid.UUID) (*T, error)

	// List, soy başına son sürümleri döndürür.
	List(ctx context.Context, scope Scope, opts ListOptions) ([]*T, error)

	// Publish, taslağı oynanabilir hâle getirir.
	Publish(ctx context.Context, scope Scope, lineageID uuid.UUID, version int) (*T, error)

	// Archive, sürümü yeni oturumlardan çeker. Mevcut referanslar geçerli
	// kalır: arşivleme, verilmiş bir sertifikayı geçersiz kılmamalı.
	Archive(ctx context.Context, scope Scope, lineageID uuid.UUID, version int) (*T, error)
}

// CharacterRepository, karakter belgeleri.
type CharacterRepository interface {
	Versioned[model.Character]
}

// ScenarioRepository, senaryo belgeleri.
type ScenarioRepository interface {
	Versioned[model.Scenario]
}

// RubricRepository, rubrik belgeleri.
type RubricRepository interface {
	Versioned[model.Rubric]
}

// LLMProfileRepository, LLM profilleri.
type LLMProfileRepository interface {
	Versioned[model.LLMProfile]

	// GetPublishedByTier, bir kademenin yayınlanmış profilini döndürür.
	//
	// Aynı kademede birden fazla yayınlanmış profil varsa en yeni sürüm
	// kazanır: yönetici yeni bir profil yayınladığında eskisini ayrıca
	// arşivlemek zorunda kalmamalı.
	GetPublishedByTier(ctx context.Context, scope Scope, tier model.Tier) (*model.LLMProfile, error)
}
