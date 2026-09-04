package graph

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/masterfabric-go/masterfabric/graph/model"
	"github.com/masterfabric-go/masterfabric/graph/scalar"
	iamDTO "github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	iamUC "github.com/masterfabric-go/masterfabric/internal/application/iam/usecase"
	auditModel "github.com/masterfabric-go/masterfabric/internal/domain/audit/model"
	iamModel "github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// mapUser, kimlik modelini şema tipine çevirir.
//
// permissions ayrı bir parametre: kullanıcı kaydında yoktur, RBAC servisinden
// gelir, ve token'daki kopyaya güvenilmez çünkü token dakikalarca eski olabilir.
func mapUser(u *iamModel.User, orgID uuid.UUID, permissions []string) *model.User {
	if u == nil {
		return nil
	}
	if permissions == nil {
		permissions = []string{}
	}
	return &model.User{
		ID:                  u.ID,
		Email:               u.Email,
		FirstName:           u.FirstName,
		LastName:            u.LastName,
		Status:              mapUserStatus(u.Status),
		EmailVerified:       u.IsEmailVerified(),
		OrganizationID:      orgID,
		Permissions:         permissions,
		DeletionRequestedAt: u.DeletionRequestedAt,
		DeletionScheduledAt: u.DeletionScheduledAt,
		CreatedAt:           u.CreatedAt,
	}
}

func mapUserStatus(s iamModel.UserStatus) model.UserStatus {
	switch s {
	case iamModel.UserStatusActive:
		return model.UserStatusActive
	case iamModel.UserStatusInactive:
		return model.UserStatusInactive
	case iamModel.UserStatusSuspended:
		return model.UserStatusSuspended
	case iamModel.UserStatusPendingDeletion:
		return model.UserStatusPendingDeletion
	case iamModel.UserStatusDeleted:
		return model.UserStatusDeleted
	default:
		return model.UserStatusInactive
	}
}

func mapPlatform(p iamModel.DevicePlatform) model.DevicePlatform {
	switch p {
	case iamModel.DevicePlatformWindows:
		return model.DevicePlatformWindows
	case iamModel.DevicePlatformMacOS:
		return model.DevicePlatformMacos
	case iamModel.DevicePlatformLinux:
		return model.DevicePlatformLinux
	case iamModel.DevicePlatformWeb:
		return model.DevicePlatformWeb
	default:
		return model.DevicePlatformUnknown
	}
}

func mapDevice(d *iamModel.Device) *model.Device {
	if d == nil {
		return nil
	}
	return &model.Device{
		ID:           d.ID,
		Name:         d.Name,
		Platform:     mapPlatform(d.Platform),
		LastSeenAt:   d.LastSeenAt,
		RevokedAt:    d.RevokedAt,
		HasPublicKey: d.PublicKey != "",
		CreatedAt:    d.CreatedAt,
	}
}

func mapDevices(devices []*iamModel.Device) []*model.Device {
	out := make([]*model.Device, 0, len(devices))
	for _, d := range devices {
		out = append(out, mapDevice(d))
	}
	return out
}

// mapDeviceInput, şema girdisini kimlik DTO'suna çevirir.
func mapDeviceInput(in *model.DeviceInput) *iamDTO.DeviceInfo {
	if in == nil {
		return nil
	}
	return &iamDTO.DeviceInfo{
		Fingerprint: in.Fingerprint,
		Name:        derefString(in.Name),
		Platform:    derefString(in.Platform),
		PublicKey:   derefString(in.PublicKey),
	}
}

// mapPairedDevice, girişte eşleşen cihazı şema tipine çevirir.
func mapPairedDevice(d *iamDTO.PairedDevice) *model.PairedDevice {
	if d == nil {
		return nil
	}
	return &model.PairedDevice{
		ID:       d.ID,
		Name:     d.Name,
		Platform: mapPlatform(iamModel.NormalizePlatform(d.Platform)),
		IsNew:    d.IsNew,
	}
}

// mapAuditEntries, denetim kayıtlarını şema tipine çevirir.
func mapAuditEntries(entries []*auditModel.AuditLog) []*model.AuditEntry {
	out := make([]*model.AuditEntry, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		entry := &model.AuditEntry{
			ID:           e.ID,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceID:   e.ResourceID,
			UserID:       e.UserID,
			CreatedAt:    e.CreatedAt,
		}
		if e.IPAddress != "" {
			ip := e.IPAddress
			entry.IPAddress = &ip
		}
		if len(e.Metadata) > 0 {
			var decoded map[string]any
			if err := json.Unmarshal(e.Metadata, &decoded); err == nil {
				entry.Metadata = scalar.JSON(decoded)
			}
		}
		out = append(out, entry)
	}
	return out
}

// authPayload, giriş sonucunu şema tipine çevirir.
//
// Kullanıcı kaydı yanıttan değil depodan okunuyor: yanıt DTO'su yaşam
// döngüsü alanlarını taşımıyor, ve me sorgusuyla giriş yanıtının aynı
// kullanıcıyı farklı göstermesi istemcide açıklanamaz bir tutarsızlık olurdu.
func (r *Resolver) authPayload(ctx context.Context, result *iamDTO.VerifyLoginCodeResponse) (*model.AuthPayload, error) {
	user, err := r.Users.GetByID(ctx, result.User.ID)
	if err != nil {
		return nil, err
	}
	return &model.AuthPayload{
		AccessToken:    result.Token,
		RefreshToken:   result.RefreshToken,
		ExpiresAt:      result.ExpiresAt,
		OrganizationID: result.OrganizationID,
		User:           mapUser(user, result.OrganizationID, r.permissionsOf(ctx, user.ID, result.OrganizationID)),
		Device:         mapPairedDevice(result.Device),
	}, nil
}

// mapDeletionStatus, silme durumunu şema tipine çevirir.
func mapDeletionStatus(s *iamUC.DeletionStatus) *model.DeletionStatus {
	if s == nil {
		return nil
	}
	return &model.DeletionStatus{
		RequestedAt: s.RequestedAt,
		ScheduledAt: s.ScheduledAt,
		Cancellable: s.Cancellable,
	}
}
