package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// SignatureVerifier, cihaz imzası doğrulayan port.
type SignatureVerifier interface {
	GenerateChallenge() (string, error)
	Verify(publicKeyB64, challenge, signatureB64 string) error
	ValidatePublicKey(publicKeyB64 string) error
}

// RequestDeviceChallengeUseCase, cihaza imzalayacağı metni verir.
//
// Challenge, kullanıcının var olup olmadığından bağımsız olarak üretilir.
// Yalnızca kayıtlı cihazlara challenge vermek, bu ucu bir "bu adres kayıtlı
// mı" oracle'ına çevirirdi — kod isteme ucunda özenle kapatılan kapının
// yanına ikinci bir kapı açmak olurdu.
type RequestDeviceChallengeUseCase struct {
	challenges repository.DeviceChallengeRepository
	verifier   SignatureVerifier
	audit      auditService.Recorder
	ttl        time.Duration
	log        *slog.Logger
	now        func() time.Time
}

// NewRequestDeviceChallengeUseCase wires the use case.
func NewRequestDeviceChallengeUseCase(
	challenges repository.DeviceChallengeRepository,
	verifier SignatureVerifier,
	audit auditService.Recorder,
	ttl time.Duration,
	log *slog.Logger,
) *RequestDeviceChallengeUseCase {
	if audit == nil {
		audit = auditService.NoopRecorder{}
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &RequestDeviceChallengeUseCase{
		challenges: challenges, verifier: verifier, audit: audit,
		ttl: ttl, log: log, now: time.Now,
	}
}

// Execute issues a challenge for the device.
func (uc *RequestDeviceChallengeUseCase) Execute(ctx context.Context, req dto.DeviceChallengeRequest) (*dto.DeviceChallengeResponse, error) {
	email := model.NormalizeEmail(req.Email)
	if email == "" || req.Fingerprint == "" {
		return nil, domainErr.New(domainErr.ErrValidation, "e-posta ve parmak izi zorunlu", nil)
	}

	challenge, err := uc.verifier.GenerateChallenge()
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "challenge üretilemedi", err)
	}

	now := uc.now().UTC()
	record := &model.DeviceChallenge{
		Email:           email,
		FingerprintHash: model.HashFingerprint(req.Fingerprint),
		Challenge:       challenge,
		ExpiresAt:       now.Add(uc.ttl),
		CreatedAt:       now,
	}
	if err := uc.challenges.Create(ctx, record); err != nil {
		return nil, err
	}

	uc.audit.Record(ctx, auditService.Entry{
		Action:       auditService.ActionDeviceChallengeIssued,
		ResourceType: "device_challenge",
		ResourceID:   record.ID.String(),
	})

	return &dto.DeviceChallengeResponse{
		Challenge: challenge,
		ExpiresAt: record.ExpiresAt,
	}, nil
}
