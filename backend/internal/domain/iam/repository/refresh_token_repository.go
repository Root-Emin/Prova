package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// RefreshTokenRepository, rotasyonlu yenileme zincirini saklar.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error

	// GetByTokenHash, hash ile halkayı bulur.
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	// MarkUsed, halkayı harcanmış işaretler.
	//
	// Yalnızca henüz kullanılmamış bir satır eşleşirse başarılı olur:
	// yeniden kullanım tespitinin tamamı buna dayanıyor, ve "önce oku sonra
	// işaretle" biçiminde yazılsaydı iki eşzamanlı istek aynı halkayı
	// harcayıp ikisi de geçerli sayılabilirdi.
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)

	// RevokeFamily, bir ailenin tüm halkalarını iptal eder ve kaç halkanın
	// etkilendiğini döndürür.
	RevokeFamily(ctx context.Context, familyID uuid.UUID, at time.Time) (int, error)

	// DeleteByUser, kalıcı silmede kullanıcının halkalarını düşürür.
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}
