package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	auditModel "github.com/masterfabric-go/masterfabric/internal/domain/audit/model"
	auditRepo "github.com/masterfabric-go/masterfabric/internal/domain/audit/repository"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	iamModel "github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// SessionAnonymizer, kalıcı silmenin nesne veritabanı ayağı.
//
// Port olarak duruyor çünkü hesap yaşam döngüsü kimlik domain'ine ait, ve
// oturum belgeleri Prova domain'ine. İkisini doğrudan bağlamak, kimlik
// katmanını eğitim içeriğine bağımlı kılardı.
type SessionAnonymizer interface {
	AnonymizeByEmployee(ctx context.Context, employeeID uuid.UUID, at time.Time) (int, error)
}

// ExportSource, dışa aktarmaya girecek oturum ve puan verisini sağlar.
type ExportSource interface {
	ExportForEmployee(ctx context.Context, employeeID uuid.UUID) (map[string]any, error)
}

// AccountDeps, hesap yaşam döngüsünün bağımlılıkları.
type AccountDeps struct {
	Users      iamRepo.UserRepository
	Devices    iamRepo.DeviceRepository
	Codes      iamRepo.LoginCodeRepository
	MagicLinks iamRepo.MagicLinkRepository
	Refresh    RefreshStore
	Audit      auditService.Recorder
	AuditRepo  auditRepo.AuditRepository
	Sessions   SessionAnonymizer
	// SessionExport, oturum ve puan verisini dışa aktarmaya ekler. Alan adı
	// metottan farklı: Go'da aynı adı taşıyan bir alan metodu gölgeler.
	SessionExport ExportSource
	// GracePeriod, silme talebi ile kalıcı silme arasındaki geri alma
	// penceresi. Env'den ayarlanabilir ki test edilebilsin.
	GracePeriod time.Duration
	Log         *slog.Logger
}

// defaultGracePeriod, geçersiz bir değer verildiğinde uygulanan pencere.
const defaultGracePeriod = 30 * 24 * time.Hour

// AccountUseCase, hesabın kayıttan kalıcı silmeye kadarki yaşam döngüsü.
type AccountUseCase struct {
	AccountDeps
	now func() time.Time
}

// NewAccountUseCase wires the use case.
func NewAccountUseCase(deps AccountDeps) *AccountUseCase {
	if deps.Audit == nil {
		deps.Audit = auditService.NoopRecorder{}
	}
	// Sıfır bekleme süresi geçerli bir değer ve bilerek korunuyor: silme
	// zincirinin uçtan uca test edilebilmesi buna bağlı. Sıfırı varsayılana
	// çevirmek, "env ile ayarlanabilsin ki test edilebilsin" gereksinimini
	// sessizce iptal ederdi. Üretimde makul bir alt sınır yapılandırma
	// doğrulamasında zorlanıyor (config.Validate).
	if deps.GracePeriod < 0 {
		deps.GracePeriod = defaultGracePeriod
	}
	return &AccountUseCase{AccountDeps: deps, now: time.Now}
}

// ProfileUpdate, kullanıcının değiştirebildiği alanlar.
type ProfileUpdate struct {
	FirstName *string
	LastName  *string
}

// UpdateProfile, ad ve soyadı günceller.
func (uc *AccountUseCase) UpdateProfile(ctx context.Context, orgID, userID uuid.UUID, update ProfileUpdate) (*iamModel.User, error) {
	user, err := uc.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDeleted() {
		return nil, domainErr.New(domainErr.ErrForbidden, "silinmiş hesap güncellenemez", nil)
	}

	if update.FirstName != nil {
		user.FirstName = strings.TrimSpace(*update.FirstName)
	}
	if update.LastName != nil {
		user.LastName = strings.TrimSpace(*update.LastName)
	}
	if err := uc.Users.Update(ctx, user); err != nil {
		return nil, err
	}

	actor := userID
	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        orgID,
		UserID:       &actor,
		Action:       auditService.ActionProfileUpdated,
		ResourceType: "user",
		ResourceID:   userID.String(),
	})
	return user, nil
}

// DeletionStatus, silme talebinin durumu.
type DeletionStatus struct {
	RequestedAt *time.Time
	ScheduledAt *time.Time
	Cancellable bool
}

// Status, hesabın silme durumunu döndürür.
func (uc *AccountUseCase) Status(ctx context.Context, userID uuid.UUID) (*DeletionStatus, error) {
	user, err := uc.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &DeletionStatus{
		RequestedAt: user.DeletionRequestedAt,
		ScheduledAt: user.DeletionScheduledAt,
		// Kalıcı silme çalıştıktan sonra geri alınamaz: geri alınacak bir
		// kişisel veri kalmamıştır.
		Cancellable: user.DeletionRequestedAt != nil && !user.IsDeleted(),
	}, nil
}

// RequestDeletion, silme talebini kaydeder ve geri alma penceresini başlatır.
//
// Hesap bu süre boyunca çalışmaya devam eder. Erişimi hemen kesmek, fikrini
// değiştiren kullanıcıyı geri alma düğmesine ulaşamaz hâle getirirdi.
func (uc *AccountUseCase) RequestDeletion(ctx context.Context, orgID, userID uuid.UUID) (*DeletionStatus, error) {
	user, err := uc.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDeleted() {
		return nil, domainErr.New(domainErr.ErrConflict, "hesap zaten silinmiş", nil)
	}

	now := uc.now().UTC()
	scheduled := now.Add(uc.GracePeriod)

	// Talep zaten varsa tarih ilerletilmiyor: ikinci kez "sil" demek,
	// bekleme süresini baştan başlatarak silmeyi geciktirmemeli.
	if user.DeletionRequestedAt == nil {
		user.DeletionRequestedAt = &now
		user.DeletionScheduledAt = &scheduled
		user.Status = iamModel.UserStatusPendingDeletion
		if err := uc.Users.Update(ctx, user); err != nil {
			return nil, err
		}
	}

	actor := userID
	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        orgID,
		UserID:       &actor,
		Action:       auditService.ActionDeletionRequeste,
		ResourceType: "user",
		ResourceID:   userID.String(),
		Metadata:     map[string]any{"scheduled_at": user.DeletionScheduledAt},
	})

	return &DeletionStatus{
		RequestedAt: user.DeletionRequestedAt,
		ScheduledAt: user.DeletionScheduledAt,
		Cancellable: true,
	}, nil
}

// CancelDeletion, silme talebini geri alır.
func (uc *AccountUseCase) CancelDeletion(ctx context.Context, orgID, userID uuid.UUID) (*DeletionStatus, error) {
	user, err := uc.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDeleted() {
		return nil, domainErr.New(domainErr.ErrConflict,
			"kalıcı silme tamamlandı; geri alınacak kişisel veri kalmadı", nil)
	}
	if user.DeletionRequestedAt == nil {
		return &DeletionStatus{Cancellable: false}, nil
	}

	user.DeletionRequestedAt = nil
	user.DeletionScheduledAt = nil
	user.Status = iamModel.UserStatusActive
	if err := uc.Users.Update(ctx, user); err != nil {
		return nil, err
	}

	actor := userID
	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        orgID,
		UserID:       &actor,
		Action:       auditService.ActionDeletionCanceled,
		ResourceType: "user",
		ResourceID:   userID.String(),
	})

	return &DeletionStatus{Cancellable: false}, nil
}

// Export, KVKK erişim hakkının karşılığı: kullanıcıya ait her şey tek belgede.
func (uc *AccountUseCase) Export(ctx context.Context, orgID, userID uuid.UUID) (map[string]any, error) {
	user, err := uc.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	devices, err := uc.Devices.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	deviceRows := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		deviceRows = append(deviceRows, map[string]any{
			"id":           d.ID,
			"name":         d.Name,
			"platform":     d.Platform,
			"last_seen_at": d.LastSeenAt,
			"revoked_at":   d.RevokedAt,
			"created_at":   d.CreatedAt,
			// Parmak izi ve açık anahtar dışarı verilmiyor: ikisi de
			// kullanıcının kendi verisi değil, cihazın kimlik bilgisi.
			// Dışa aktarma dosyası paylaşılabilir bir belge ve içine
			// çalışan bir kimlik bilgisi koymak, dosyayı bir anahtarlığa
			// çevirirdi.
		})
	}

	auditRows, _, err := uc.AuditRepo.ListByUser(ctx, userID, 0, maxExportAuditRows)
	if err != nil {
		return nil, err
	}

	export := map[string]any{
		"exported_at": uc.now().UTC(),
		"profile": map[string]any{
			"id":                    user.ID,
			"email":                 user.Email,
			"first_name":            user.FirstName,
			"last_name":             user.LastName,
			"status":                user.Status,
			"email_verified_at":     user.EmailVerifiedAt,
			"deletion_requested_at": user.DeletionRequestedAt,
			"created_at":            user.CreatedAt,
		},
		"devices":   deviceRows,
		"audit_log": exportAuditRows(auditRows),
	}

	// Oturumlar, transkriptler ve puanlar nesne veritabanından geliyor.
	// Bağlanmamışsa dışa aktarma yine üretiliyor ama eksik olduğu açıkça
	// belirtiliyor: sessizce eksik bir KVKK yanıtı, yanlış bir yanıttır.
	if uc.SessionExport != nil {
		sessions, err := uc.SessionExport.ExportForEmployee(ctx, userID)
		if err != nil {
			return nil, err
		}
		for key, value := range sessions {
			export[key] = value
		}
	} else {
		export["sessions_unavailable"] = "nesne veritabanı bağlı değil"
	}

	actor := userID
	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        orgID,
		UserID:       &actor,
		Action:       auditService.ActionDataExported,
		ResourceType: "user",
		ResourceID:   userID.String(),
	})

	return export, nil
}

// maxExportAuditRows, dışa aktarmaya girecek denetim kaydı üst sınırı.
const maxExportAuditRows = 5000

func exportAuditRows(entries []*auditModel.AuditLog) []map[string]any {
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, map[string]any{
			"action":        e.Action,
			"resource_type": e.ResourceType,
			"resource_id":   e.ResourceID,
			"ip_address":    e.IPAddress,
			"created_at":    e.CreatedAt,
		})
	}
	return rows
}

// PurgeResult, tek bir kalıcı silme işinin sonucu.
type PurgeResult struct {
	Purged             int
	AnonymizedSessions int
}

// PurgeDue, geri alma penceresi dolmuş hesapları kalıcı olarak siler.
//
// İki veritabanında birden çalışır ve ikisinde farklı davranır:
// PostgreSQL'de kişisel veri gerçekten silinir; MongoDB'de oturumlar
// silinmez, kimliksizleştirilir. Sebep KVKK'nın iki ayrı ilkesi: silme
// hakkı kişisel veriye uygulanır, ama kurumun eğitim etkinliği ölçümü
// kişisel veri değildir ve silinen bir çalışan yüzünden geriye dönük
// bozulmamalıdır.
func (uc *AccountUseCase) PurgeDue(ctx context.Context, limit int) (PurgeResult, error) {
	now := uc.now().UTC()

	due, err := uc.Users.ListDuePurge(ctx, now, limit)
	if err != nil {
		return PurgeResult{}, err
	}

	var result PurgeResult
	for _, user := range due {
		anonymized, err := uc.purgeOne(ctx, user, now)
		if err != nil {
			// Bir hesabın silinememesi diğerlerini engellememeli; iş bir
			// sonraki koşuda kaldığı yerden devam eder.
			uc.Log.ErrorContext(ctx, "hesap silinemedi", "user_id", user.ID, "error", err)
			continue
		}
		result.Purged++
		result.AnonymizedSessions += anonymized
	}
	return result, nil
}

// purgeOne, tek bir hesabı siler.
//
// Sıra önemli: önce nesne veritabanı kimliksizleştirilir, sonra ilişkili
// kimlik satırları düşürülür, en son kullanıcı satırı silinmiş işaretlenir.
// Ters sırada yapılsaydı ve arada bir hata olsaydı, kullanıcı "silindi"
// görünürken oturumları hâlâ kimliğine bağlı kalırdı.
func (uc *AccountUseCase) purgeOne(ctx context.Context, user *iamModel.User, now time.Time) (int, error) {
	var anonymized int
	if uc.Sessions != nil {
		count, err := uc.Sessions.AnonymizeByEmployee(ctx, user.ID, now)
		if err != nil {
			return 0, err
		}
		anonymized = count
	}

	if uc.Refresh != nil {
		if err := uc.Refresh.DeleteByUser(ctx, user.ID); err != nil {
			return 0, err
		}
	}
	if uc.MagicLinks != nil {
		if err := uc.MagicLinks.DeleteByUser(ctx, user.ID.String()); err != nil {
			return 0, err
		}
	}
	if user.Email != "" && uc.Codes != nil {
		if err := uc.Codes.DeleteByEmail(ctx, user.Email); err != nil {
			return 0, err
		}
	}
	if err := uc.Devices.DeleteByUser(ctx, user.ID); err != nil {
		return 0, err
	}

	if err := uc.Users.Purge(ctx, user.ID, now); err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			// Başka bir koşu önce davrandı; bu bir hata değil.
			return anonymized, nil
		}
		return 0, err
	}

	// Silme olayının kendisi denetime düşer ve denetim kaydı silinmez.
	// KVKK hesap verebilirlik ilkesi bunu gerektiriyor: silindiğini
	// kanıtlayamayan bir silme, yapılmamış bir silmeden ayırt edilemez.
	actor := user.ID
	uc.Audit.Record(ctx, auditService.Entry{
		UserID:       &actor,
		Action:       auditService.ActionAccountPurged,
		ResourceType: "user",
		ResourceID:   user.ID.String(),
		Metadata: map[string]any{
			"anonymized_sessions": anonymized,
			"purged_at":           now,
		},
	})

	return anonymized, nil
}

// RefreshStore, kalıcı silmede refresh token satırlarını düşürür.
type RefreshStore interface {
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}
