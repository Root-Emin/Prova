// Package audit, denetim kaydı portunun PostgreSQL arkalı uygulamasıdır.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/audit/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/audit/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
)

// writeTimeout, tek bir denetim yazımının üst sınırı. İstek bağlamı çoktan
// bitmiş olabileceği için yazım kendi bağlamıyla çalışır.
const writeTimeout = 5 * time.Second

// Recorder, olayları audit_logs tablosuna yazar.
//
// Yazım eşzamansızdır ama takip edilir: Wait, kapanış sırasında uçuştaki
// yazımların tamamlanmasını bekler. "Best effort" olması kaydın kaybolmasını
// meşrulaştırmaz — yalnızca kaydın kullanıcıyı bekletmemesini sağlar.
type Recorder struct {
	repo repository.AuditRepository
	log  *slog.Logger
	wg   sync.WaitGroup
}

// NewRecorder wires the recorder.
func NewRecorder(repo repository.AuditRepository, log *slog.Logger) *Recorder {
	return &Recorder{repo: repo, log: log}
}

// Record implements service.Recorder.
func (r *Recorder) Record(ctx context.Context, entry service.Entry) {
	if r == nil || r.repo == nil {
		return
	}

	row := &model.AuditLog{
		OrganizationID: entry.OrgID,
		UserID:         entry.UserID,
		RequestID:      entry.RequestID,
		Action:         string(entry.Action),
		ResourceType:   entry.ResourceType,
		ResourceID:     entry.ResourceID,
		IPAddress:      entry.IPAddress,
		UserAgent:      entry.UserAgent,
	}
	if len(entry.Metadata) > 0 {
		if encoded, err := json.Marshal(entry.Metadata); err == nil {
			row.Metadata = encoded
		}
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), writeTimeout)
		defer cancel()
		if err := r.repo.Create(writeCtx, row); err != nil {
			r.log.ErrorContext(writeCtx, "denetim kaydı yazılamadı",
				"action", row.Action,
				"resource_type", row.ResourceType,
				"error", err,
			)
		}
	}()
}

// Wait, uçuştaki denetim yazımlarının bitmesini bekler. Kapanış yolunda
// çağrılır; aksi hâlde son işlemin kaydı süreç ölürken kaybolur.
func (r *Recorder) Wait() {
	if r == nil {
		return
	}
	r.wg.Wait()
}
