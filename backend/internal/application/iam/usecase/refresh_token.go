package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// TokenMinter, access token'ı verilen ömürle üretir.
type TokenMinter interface {
	GenerateTokenWithTTL(ctx context.Context, claims service.TokenClaims, ttl time.Duration) (string, error)
}

// RefreshDeps, yenileme akışının bağımlılıkları.
type RefreshDeps struct {
	Tokens     repository.RefreshTokenRepository
	Users      repository.UserRepository
	Devices    repository.DeviceRepository
	Minter     TokenMinter
	Hasher     MagicLinkTokens
	Denylist   FamilyRevoker
	Audit      auditService.Recorder
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Log        *slog.Logger
}

// FamilyRevoker, iptal edilen aileyi taşıma katmanına bildirir.
type FamilyRevoker interface {
	RevokeFamily(ctx context.Context, familyID uuid.UUID, until time.Time) error
	IsFamilyRevoked(ctx context.Context, familyID uuid.UUID) (bool, error)
}

// RefreshTokenUseCase, access token'ı rotasyonla yeniler.
type RefreshTokenUseCase struct {
	RefreshDeps
	now func() time.Time
}

// NewRefreshTokenUseCase wires the use case.
func NewRefreshTokenUseCase(deps RefreshDeps) *RefreshTokenUseCase {
	if deps.Audit == nil {
		deps.Audit = auditService.NoopRecorder{}
	}
	return &RefreshTokenUseCase{RefreshDeps: deps, now: time.Now}
}

// errInvalidRefresh, her reddin toplandığı tek cevap.
func errInvalidRefresh() error {
	return domainErr.New(domainErr.ErrUnauthorized, "geçersiz veya süresi dolmuş yenileme token'ı", nil)
}

// Issue, giriş sonrası ilk refresh token'ı üretir.
func (uc *RefreshTokenUseCase) Issue(ctx context.Context, userID, orgID uuid.UUID, deviceID *uuid.UUID) (token string, familyID uuid.UUID, err error) {
	raw, hash, err := uc.Hasher.Generate()
	if err != nil {
		return "", uuid.Nil, domainErr.New(domainErr.ErrInternal, "yenileme token'ı üretilemedi", err)
	}

	now := uc.now().UTC()
	record := &model.RefreshToken{
		ID:             uuid.New(),
		UserID:         userID,
		TokenHash:      hash,
		DeviceID:       deviceID,
		OrganizationID: orgID,
		ExpiresAt:      now.Add(uc.RefreshTTL),
		CreatedAt:      now,
	}
	// Ailenin kimliği ilk halkanın kimliği: aile ayrı bir kayıt olmadan
	// izlenebilir kalır ve zincirin kökü her zaman bellidir.
	record.FamilyID = record.ID

	if err := uc.Tokens.Create(ctx, record); err != nil {
		return "", uuid.Nil, err
	}
	return raw, record.FamilyID, nil
}

// Execute, refresh token'ı yeni bir çifte çevirir.
//
// Her kullanımda rotasyon: eski halka harcanır, yerine yenisi verilir.
// Rotasyon olmadan çalınmış bir refresh token, ömrü boyunca sınırsız access
// token üretirdi ve bunu fark etmenin hiçbir yolu olmazdı.
func (uc *RefreshTokenUseCase) Execute(ctx context.Context, req dto.RefreshTokenRequest) (*dto.VerifyLoginCodeResponse, error) {
	if req.RefreshToken == "" {
		return nil, errInvalidRefresh()
	}
	now := uc.now().UTC()

	record, err := uc.Tokens.GetByTokenHash(ctx, uc.Hasher.Hash(req.RefreshToken))
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, errInvalidRefresh()
		}
		return nil, err
	}

	// Yeniden kullanım tespiti. Bu, akıştaki en önemli tek karar: harcanmış
	// bir halkanın tekrar gelmesi ya token'ın çalındığı ya da eski bir
	// kopyanın oynatıldığı anlamına gelir, ve ikisini ayırt etmek mümkün
	// olmadığı için ailenin tamamı kesilir.
	if record.IsReplay(now) {
		uc.killFamily(ctx, record, now, "reuse_detected")
		return nil, errInvalidRefresh()
	}
	if !record.IsUsable(now) {
		return nil, errInvalidRefresh()
	}

	// Aile daha önce iptal edilmişse (başka bir yeniden kullanım ya da
	// çıkış), bu halka geçerli görünse bile kabul edilmemeli.
	if uc.Denylist != nil {
		revoked, err := uc.Denylist.IsFamilyRevoked(ctx, record.FamilyID)
		if err != nil {
			uc.Log.ErrorContext(ctx, "aile iptal listesi okunamadı", "error", err)
		} else if revoked {
			return nil, errInvalidRefresh()
		}
	}

	used, err := uc.Tokens.MarkUsed(ctx, record.ID, now)
	if err != nil {
		return nil, err
	}
	if !used {
		// Yarışı kaybettik: aynı halka başka bir istek tarafından harcandı.
		// Bu da bir yeniden kullanım belirtisi.
		uc.killFamily(ctx, record, now, "concurrent_use")
		return nil, errInvalidRefresh()
	}

	user, err := uc.Users.GetByID(ctx, record.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() && !user.IsPendingDeletion() {
		return nil, domainErr.New(domainErr.ErrForbidden, "hesap etkin değil", nil)
	}

	// Cihaz iptal edilmişse yenileme de durmalı: cihaz iptali, o cihazdaki
	// oturumun sona ermesi demek.
	if record.DeviceID != nil && uc.Devices != nil {
		device, err := uc.Devices.GetByID(ctx, record.UserID, *record.DeviceID)
		if err == nil && device.IsRevoked() {
			return nil, domainErr.New(domainErr.ErrForbidden, "bu cihazın yetkisi iptal edilmiş", nil)
		}
	}

	rotated, err := uc.rotate(ctx, record, now)
	if err != nil {
		return nil, err
	}

	accessToken, err := uc.Minter.GenerateTokenWithTTL(ctx, service.TokenClaims{
		UserID:          user.ID,
		Email:           user.Email,
		OrganizationID:  record.OrganizationID,
		DeviceID:        record.DeviceID,
		RefreshFamilyID: &record.FamilyID,
	}, uc.AccessTTL)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "access token üretilemedi", err)
	}

	userID := user.ID
	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        record.OrganizationID,
		UserID:       &userID,
		Action:       auditService.ActionRefreshRotated,
		ResourceType: "refresh_token",
		ResourceID:   record.FamilyID.String(),
	})

	return &dto.VerifyLoginCodeResponse{
		Token:          accessToken,
		RefreshToken:   rotated,
		ExpiresAt:      now.Add(uc.AccessTTL),
		OrganizationID: record.OrganizationID,
		User: dto.UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

// rotate, zincire yeni bir halka ekler.
func (uc *RefreshTokenUseCase) rotate(ctx context.Context, previous *model.RefreshToken, now time.Time) (string, error) {
	raw, hash, err := uc.Hasher.Generate()
	if err != nil {
		return "", domainErr.New(domainErr.ErrInternal, "yenileme token'ı üretilemedi", err)
	}

	parentID := previous.ID
	next := &model.RefreshToken{
		ID:             uuid.New(),
		UserID:         previous.UserID,
		FamilyID:       previous.FamilyID,
		TokenHash:      hash,
		DeviceID:       previous.DeviceID,
		OrganizationID: previous.OrganizationID,
		ParentID:       &parentID,
		// Ailenin toplam ömrü uzatılmıyor: yenileme, oturumu sonsuza kadar
		// uzatan bir mekanizma olmamalı.
		ExpiresAt: previous.ExpiresAt,
		CreatedAt: now,
	}
	if err := uc.Tokens.Create(ctx, next); err != nil {
		return "", err
	}
	return raw, nil
}

// killFamily, aileyi iptal eder ve olayı denetime yazar.
func (uc *RefreshTokenUseCase) killFamily(ctx context.Context, record *model.RefreshToken, now time.Time, reason string) {
	revoked, err := uc.Tokens.RevokeFamily(ctx, record.FamilyID, now)
	if err != nil {
		uc.Log.ErrorContext(ctx, "refresh ailesi iptal edilemedi",
			"family_id", record.FamilyID, "error", err)
	}
	if uc.Denylist != nil {
		if err := uc.Denylist.RevokeFamily(ctx, record.FamilyID, record.ExpiresAt); err != nil {
			uc.Log.ErrorContext(ctx, "aile iptal listesine yazılamadı",
				"family_id", record.FamilyID, "error", err)
		}
	}

	userID := record.UserID
	uc.Audit.Record(ctx, auditService.Entry{
		OrgID:        record.OrganizationID,
		UserID:       &userID,
		Action:       auditService.ActionRefreshReuse,
		ResourceType: "refresh_token",
		ResourceID:   record.FamilyID.String(),
		Metadata: map[string]any{
			"reason":         reason,
			"revoked_tokens": revoked,
			"replayed_token": record.ID.String(),
		},
	})

	uc.Log.WarnContext(ctx, "refresh token yeniden kullanımı: aile iptal edildi",
		"family_id", record.FamilyID, "user_id", record.UserID, "reason", reason)
}
