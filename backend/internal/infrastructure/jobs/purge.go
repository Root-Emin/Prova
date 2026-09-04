// Package jobs, zamanlanmış arka plan işlerini barındırır.
package jobs

import (
	"context"
	"log/slog"
	"time"

	iamUC "github.com/masterfabric-go/masterfabric/internal/application/iam/usecase"
)

// purgeBatchSize, tek koşuda silinecek hesap sayısı.
//
// Sınırlı: bir koşuda binlerce hesabı silmek, iki veritabanına aynı anda
// uzun süreli yük bindirir. Kalanlar bir sonraki koşuda alınır ve silme
// zaten geri alma penceresi dolduktan sonra çalışıyor — birkaç dakikalık
// gecikmenin bir maliyeti yok.
const purgeBatchSize = 100

// PurgeJob, geri alma penceresi dolmuş hesapları kalıcı olarak siler.
type PurgeJob struct {
	accounts *iamUC.AccountUseCase
	interval time.Duration
	log      *slog.Logger
}

// NewPurgeJob wires the job.
func NewPurgeJob(accounts *iamUC.AccountUseCase, interval time.Duration, log *slog.Logger) *PurgeJob {
	if interval <= 0 {
		interval = time.Hour
	}
	return &PurgeJob{accounts: accounts, interval: interval, log: log}
}

// Start runs the job until the context is cancelled.
//
// İlk koşu hemen yapılıyor: sunucu yeniden başlatıldığında, bekleyen bir
// silmenin bir sonraki aralığa kadar ertelenmesi için bir neden yok.
func (j *PurgeJob) Start(ctx context.Context) {
	j.runOnce(ctx)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.log.Info("kalıcı silme işi durdu")
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

// RunOnce, işi bir kez çalıştırır. Doğrulama ve duman testleri bekleme
// süresini beklemeden silmeyi tetikleyebilsin diye dışa açık.
func (j *PurgeJob) RunOnce(ctx context.Context) (iamUC.PurgeResult, error) {
	return j.accounts.PurgeDue(ctx, purgeBatchSize)
}

func (j *PurgeJob) runOnce(ctx context.Context) {
	result, err := j.RunOnce(ctx)
	if err != nil {
		j.log.ErrorContext(ctx, "kalıcı silme işi başarısız", "error", err)
		return
	}
	if result.Purged > 0 {
		j.log.InfoContext(ctx, "kalıcı silme tamamlandı",
			"purged", result.Purged, "anonymized_sessions", result.AnonymizedSessions)
	}
}
