package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*model.User, int, error)

	// ListDuePurge, geri alma penceresi dolmuş ve henüz silinmemiş
	// hesapları döndürür. Zamanlanmış silme işi bunu kullanır.
	ListDuePurge(ctx context.Context, now time.Time, limit int) ([]*model.User, error)

	// Purge, kişisel veriyi gerçekten siler ama satırı bırakır.
	//
	// Satır kalıyor çünkü denetim kaydı silinmez ve bir kullanıcı kimliğine
	// bağlanabilmelidir. E-posta ve isim null'a çekilir, durum "deleted"
	// olur; geriye kimliksiz bir kabuk kalır.
	Purge(ctx context.Context, id uuid.UUID, at time.Time) error

	// RecordFailedAttempt, başarısız giriş sayacını artırır ve eşiği
	// aşarsa hesabı geçici olarak kilitler. Güncellenmiş sayacı döndürür.
	RecordFailedAttempt(ctx context.Context, id uuid.UUID, threshold int, lockUntil time.Time) (int, error)

	// ClearFailedAttempts, başarılı girişten sonra sayacı ve kilidi sıfırlar.
	ClearFailedAttempts(ctx context.Context, id uuid.UUID) error
}
