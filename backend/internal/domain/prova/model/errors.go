package model

import (
	"time"

	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// nowFunc, test edilebilirlik için saat kaynağı.
type nowFunc func() time.Time

// Doğrulama hataları.
//
// Her biri ErrValidation'ı sarmalıyor, böylece taşıma katmanı tek bir
// errors.Is ile 422 üretebiliyor ve mesaj kullanıcıya olduğu gibi gidebiliyor.
var (
	ErrCharacterNameRequired      = domainErr.New(domainErr.ErrValidation, "karakter adı zorunlu", nil)
	ErrCharacterPersonaRequired   = domainErr.New(domainErr.ErrValidation, "karakter tanımı zorunlu", nil)
	ErrCharacterDifficultyInvalid = domainErr.New(domainErr.ErrValidation, "geçersiz zorluk seviyesi", nil)

	ErrScenarioTitleRequired     = domainErr.New(domainErr.ErrValidation, "senaryo başlığı zorunlu", nil)
	ErrScenarioContextRequired   = domainErr.New(domainErr.ErrValidation, "senaryo bağlamı zorunlu", nil)
	ErrScenarioCharacterRequired = domainErr.New(domainErr.ErrValidation, "senaryo bir karaktere bağlanmalı", nil)
	ErrScenarioRubricRequired    = domainErr.New(domainErr.ErrValidation, "senaryo bir rubriğe bağlanmalı", nil)

	ErrRubricNameRequired     = domainErr.New(domainErr.ErrValidation, "rubrik adı zorunlu", nil)
	ErrRubricCriteriaRequired = domainErr.New(domainErr.ErrValidation, "rubrik en az bir kriter içermeli", nil)
	ErrRubricCriterionKeyDup  = domainErr.New(domainErr.ErrValidation, "kriter anahtarları benzersiz olmalı", nil)
	ErrRubricThresholdInvalid = domainErr.New(domainErr.ErrValidation, "geçme eşiği 0 ile 1 arasında olmalı", nil)

	ErrLLMProfileModelRequired    = domainErr.New(domainErr.ErrValidation, "model kimliği zorunlu", nil)
	ErrLLMProfileBaseURLRequired  = domainErr.New(domainErr.ErrValidation, "sağlayıcı base URL'i zorunlu", nil)
	ErrLLMProfileTierInvalid      = domainErr.New(domainErr.ErrValidation, "geçersiz kademe", nil)
	ErrLLMProfileSamplingInvalid  = domainErr.New(domainErr.ErrValidation, "sıcaklık ve topP 0 ile 2 arasında olmalı", nil)
	ErrLLMProfileMaxTokensInvalid = domainErr.New(domainErr.ErrValidation, "maksimum token pozitif olmalı", nil)
)

// Sürümleme hataları.
var (
	// ErrDocumentFrozen, yayınlanmış ya da arşivlenmiş bir belgeyi yerinde
	// güncelleme denemesinin karşılığı. Sürümlemenin tüm anlamı buna bağlı:
	// yayınlanmış belge değişebilseydi, bir sertifikanın hangi metinle
	// verildiği sonradan yeniden yazılabilirdi.
	ErrDocumentFrozen = domainErr.New(domainErr.ErrConflict,
		"yayınlanmış belge güncellenemez; düzenleme yeni sürüm yaratır", nil)
	// ErrDocumentNotDraft, taslak olmayan bir belgeyi yayınlama denemesi.
	ErrDocumentNotDraft = domainErr.New(domainErr.ErrConflict, "yalnızca taslak sürüm yayınlanabilir", nil)
	// ErrDocumentNotFound, verilen soy ya da sürüm bu organizasyonda yok.
	ErrDocumentNotFound = domainErr.New(domainErr.ErrNotFound, "belge bulunamadı", nil)
	// ErrOrgRequired, kiracı sınırı olmadan sorgu kurma denemesi.
	ErrOrgRequired = domainErr.New(domainErr.ErrValidation,
		"organizasyon kimliği zorunlu; kiracı sınırı olmadan sorgu çalıştırılamaz", nil)
)

// Oturum hataları.
var (
	ErrSessionNotActive     = domainErr.New(domainErr.ErrConflict, "oturum aktif değil", nil)
	ErrSessionTurnLimit     = domainErr.New(domainErr.ErrConflict, "oturumun konuşma sırası limiti doldu", nil)
	ErrSessionNotOwned      = domainErr.New(domainErr.ErrForbidden, "bu oturum size ait değil", nil)
	ErrSessionAlreadyScored = domainErr.New(domainErr.ErrConflict, "oturum zaten puanlandı", nil)
)
