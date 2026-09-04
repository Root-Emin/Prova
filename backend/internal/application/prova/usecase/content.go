package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	provaModel "github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	provaRepo "github.com/masterfabric-go/masterfabric/internal/domain/prova/repository"
)

// versionedDoc, generic içerik akışının belgeye koyduğu kısıt.
type versionedDoc[T any] interface {
	*T
	provaModel.Versioned
}

// ContentUseCase, sürümlenebilir bir belge türünün yazma akışı.
//
// Generic: karakter, senaryo, rubrik ve LLM profili aynı üç işlemi paylaşıyor
// (oluştur, düzenle, yayınla) ve aynı sürümleme kurallarına uyuyor. Dört kez
// yazmak, kuralların dördünde ayrı ayrı bozulabilmesi demek olurdu.
type ContentUseCase[T any, PT versionedDoc[T]] struct {
	repo  provaRepo.Versioned[T]
	audit auditService.Recorder
	// resource, denetim kaydındaki kaynak türü ("character", "rubric"...).
	resource string
	now      func() time.Time
}

// NewContentUseCase wires the use case.
func NewContentUseCase[T any, PT versionedDoc[T]](
	repo provaRepo.Versioned[T],
	audit auditService.Recorder,
	resource string,
) *ContentUseCase[T, PT] {
	if audit == nil {
		audit = auditService.NoopRecorder{}
	}
	return &ContentUseCase[T, PT]{repo: repo, audit: audit, resource: resource, now: time.Now}
}

// Create, yeni bir soyun ilk taslağını yazar.
func (uc *ContentUseCase[T, PT]) Create(ctx context.Context, scope provaRepo.Scope, userID uuid.UUID, apply func(PT)) (*T, error) {
	var doc T
	pointer := PT(&doc)

	envelope := pointer.Envelope()
	*envelope = provaModel.NewLineage(scope.OrgID(), userID, uc.now())

	apply(pointer)

	// Doğrulama yazımdan önce: geçersiz bir taslak yayınlanamaz ama depoda
	// durur, ve bir sonraki düzenlemede sessizce yeni sürümün temeli olur.
	if err := pointer.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.Create(ctx, scope, &doc); err != nil {
		return nil, err
	}

	uc.recordChange(ctx, scope, userID, auditService.ActionContentVersioned, envelope, "created")
	return &doc, nil
}

// Update, belgeyi düzenler.
//
// Son sürüm taslaksa yerinde güncellenir; yayınlanmışsa yeni sürüm yaratılır.
// Taslak, değişmezliğin tek istisnası ve istisna tam olarak şu yüzden
// geçerli: hiçbir oturum bir taslağa referans veremez.
func (uc *ContentUseCase[T, PT]) Update(ctx context.Context, scope provaRepo.Scope, userID, lineageID uuid.UUID, apply func(PT)) (*T, error) {
	latest, err := uc.repo.GetLatest(ctx, scope, lineageID)
	if err != nil {
		return nil, err
	}

	if PT(latest).Envelope().Status == provaModel.DocumentStatusDraft {
		apply(PT(latest))
		if err := PT(latest).Validate(); err != nil {
			return nil, err
		}
		if err := uc.repo.UpdateDraft(ctx, scope, latest); err != nil {
			return nil, err
		}
		uc.recordChange(ctx, scope, userID, auditService.ActionContentVersioned, PT(latest).Envelope(), "draft_updated")
		return latest, nil
	}

	var validationErr error
	next, err := uc.repo.NewVersion(ctx, scope, lineageID, userID, func(doc *T) {
		apply(PT(doc))
		validationErr = PT(doc).Validate()
	})
	if err != nil {
		return nil, err
	}
	// Doğrulama depo yazımından sonra kontrol ediliyor çünkü apply, deponun
	// sürüm zarfını kurduğu callback'in içinde çalışıyor. Geçersiz sürüm
	// taslak olarak kalır ve yayınlanamaz; kullanıcı hatayı görür.
	if validationErr != nil {
		return nil, validationErr
	}

	uc.recordChange(ctx, scope, userID, auditService.ActionContentVersioned, PT(next).Envelope(), "new_version")
	return next, nil
}

// Publish, taslağı oynanabilir hâle getirir.
func (uc *ContentUseCase[T, PT]) Publish(ctx context.Context, scope provaRepo.Scope, userID, lineageID uuid.UUID, version int) (*T, error) {
	published, err := uc.repo.Publish(ctx, scope, lineageID, version)
	if err != nil {
		return nil, err
	}
	uc.recordChange(ctx, scope, userID, auditService.ActionContentPublished, PT(published).Envelope(), "published")
	return published, nil
}

func (uc *ContentUseCase[T, PT]) recordChange(
	ctx context.Context,
	scope provaRepo.Scope,
	userID uuid.UUID,
	action auditService.Action,
	envelope *provaModel.Document,
	change string,
) {
	actor := userID
	uc.audit.Record(ctx, auditService.Entry{
		OrgID:        scope.OrgID(),
		UserID:       &actor,
		Action:       action,
		ResourceType: uc.resource,
		ResourceID:   envelope.LineageID.String(),
		Metadata: map[string]any{
			"change":     change,
			"version":    envelope.Version,
			"version_id": envelope.ID.String(),
		},
	})
}
