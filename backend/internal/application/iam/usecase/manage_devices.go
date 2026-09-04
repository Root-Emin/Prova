package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// ManageDevicesUseCase, kullanıcının kayıtlı cihazlarını listeler ve iptal eder.
type ManageDevicesUseCase struct {
	devices  repository.DeviceRepository
	denylist TokenRevoker
	audit    auditService.Recorder
	log      *slog.Logger
	now      func() time.Time
}

// TokenRevoker, iptal edilen cihaza ait token'ları anında geçersizleştirir.
//
// Cihazı iptal edip token'ını yaşamaya bırakmak, iptali bir sonraki token
// yenilemesine kadar erteler; çalınmış bir makine için bu pencere çok uzun.
type TokenRevoker interface {
	RevokeDeviceTokens(ctx context.Context, userID, deviceID uuid.UUID, until time.Time) error
}

// NewManageDevicesUseCase wires the use case.
func NewManageDevicesUseCase(
	devices repository.DeviceRepository,
	denylist TokenRevoker,
	audit auditService.Recorder,
	log *slog.Logger,
) *ManageDevicesUseCase {
	if audit == nil {
		audit = auditService.NoopRecorder{}
	}
	return &ManageDevicesUseCase{devices: devices, denylist: denylist, audit: audit, log: log, now: time.Now}
}

// List returns every device paired with the user, revoked ones included.
//
// İptal edilenler de döner: kullanıcı hangi makineyi ne zaman iptal ettiğini
// görebilmeli, yoksa iptal işlemi kanıtsız kalır.
func (uc *ManageDevicesUseCase) List(ctx context.Context, userID uuid.UUID) ([]*model.Device, error) {
	return uc.devices.ListByUser(ctx, userID)
}

// Revoke withdraws a device's authorisation and kills its live tokens.
func (uc *ManageDevicesUseCase) Revoke(ctx context.Context, orgID, userID, deviceID uuid.UUID) error {
	now := uc.now().UTC()

	// Sahiplik depo katmanında zorlanıyor: userID filtresi olmadan bir
	// kullanıcı başkasının cihazını iptal edebilirdi.
	if err := uc.devices.Revoke(ctx, userID, deviceID, now); err != nil {
		return err
	}

	if uc.denylist != nil {
		// Denylist yazımı başarısız olursa iptal yarım kalmış olur: cihaz
		// kayıtta iptal ama token'ı hâlâ geçerli. Bu, sessizce geçilemeyecek
		// bir durum.
		if err := uc.denylist.RevokeDeviceTokens(ctx, userID, deviceID, now.Add(maxTokenLifetime)); err != nil {
			uc.log.ErrorContext(ctx, "cihaz token'ları iptal listesine yazılamadı",
				"device_id", deviceID, "error", err)
			return domainErr.New(domainErr.ErrInternal,
				"cihaz iptal edildi ama açık oturumları kapatılamadı", err)
		}
	}

	uc.audit.Record(ctx, auditService.Entry{
		OrgID:        orgID,
		UserID:       &userID,
		Action:       auditService.ActionDeviceRevoked,
		ResourceType: "device",
		ResourceID:   deviceID.String(),
	})
	return nil
}

// maxTokenLifetime, denylist girdisinin saklanma süresinin üst sınırı.
// Access token'ın ömrü bundan kısa olduğu için, bu süreden sonra girdiyi
// tutmanın hiçbir etkisi kalmaz.
const maxTokenLifetime = 24 * time.Hour
