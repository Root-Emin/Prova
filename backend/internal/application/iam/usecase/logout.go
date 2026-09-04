package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
)

// LogoutUseCase, oturumu her iki ucundan da kapatır.
//
// Yalnızca istemciden token'ı silmek çıkış değildir: imzalı token sona erene
// kadar geçerli kalır ve refresh zinciri yeni access token üretmeye devam
// eder. Çıkış, aktif token'ı iptal listesine almak ve refresh ailesini
// öldürmektir.
type LogoutUseCase struct {
	sessions SessionKiller
	audit    auditService.Recorder
	log      *slog.Logger
	now      func() time.Time
	// accessTTL, iptal listesi girdisinin ne kadar tutulacağını belirler.
	// Token zaten sona erdikten sonra listede tutmanın etkisi yok.
	accessTTL time.Duration
}

// SessionKiller, oturumu sonlandıran iptal işlemlerini toplar.
type SessionKiller interface {
	// RevokeToken, tek bir access token'ı reddeder.
	RevokeToken(ctx context.Context, tokenID string, until time.Time) error
	// RevokeFamily, bir refresh token ailesini reddeder.
	RevokeFamily(ctx context.Context, familyID uuid.UUID, until time.Time) error
}

// NewLogoutUseCase wires the use case.
func NewLogoutUseCase(
	sessions SessionKiller,
	audit auditService.Recorder,
	accessTTL time.Duration,
	log *slog.Logger,
) *LogoutUseCase {
	if audit == nil {
		audit = auditService.NoopRecorder{}
	}
	return &LogoutUseCase{sessions: sessions, audit: audit, accessTTL: accessTTL, log: log, now: time.Now}
}

// Execute kills the caller's session.
func (uc *LogoutUseCase) Execute(ctx context.Context, orgID, userID uuid.UUID, tokenID string, familyID *uuid.UUID) error {
	now := uc.now().UTC()

	if uc.sessions != nil {
		if tokenID != "" {
			if err := uc.sessions.RevokeToken(ctx, tokenID, now.Add(uc.accessTTL)); err != nil {
				return err
			}
		}
		if familyID != nil {
			// Aile iptali, access token'ın sona ermesini beklemeden refresh
			// zincirini keser. Süre olarak refresh ömrü değil, kabaca üst
			// sınır kullanılıyor; girdi sona erdiğinde aile zaten ölü.
			if err := uc.sessions.RevokeFamily(ctx, *familyID, now.Add(refreshDenylistTTL)); err != nil {
				return err
			}
		}
	}

	uc.audit.Record(ctx, auditService.Entry{
		OrgID:        orgID,
		UserID:       &userID,
		Action:       auditService.ActionLogout,
		ResourceType: "session",
		ResourceID:   tokenID,
	})
	return nil
}

// refreshDenylistTTL, iptal edilmiş bir refresh ailesinin listede kalma süresi.
// Refresh token'ın azami ömründen uzun olmalı ki iptal, token sona ermeden
// önce unutulmasın.
const refreshDenylistTTL = 90 * 24 * time.Hour
