package repository

import (
	"context"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// MagicLinkRepository, tek kullanımlık giriş bağlantılarını saklar.
type MagicLinkRepository interface {
	Create(ctx context.Context, link *model.MagicLink) error

	// GetByTokenHash, hash ile bağlantıyı bulur. Token metniyle arama yok:
	// depoda token metni hiç yok.
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.MagicLink, error)

	// Consume, bağlantıyı tek kullanımlık olarak yakar.
	//
	// Yalnızca henüz kullanılmamış bir satır eşleşirse başarılı olur:
	// "önce oku sonra işaretle" biçiminde yazılsaydı, aynı bağlantıya
	// gelen iki eşzamanlı istek iki oturum üretebilirdi.
	Consume(ctx context.Context, id string, at time.Time) (bool, error)

	// InvalidateActive, adrese ait kullanılmamış bağlantıları iptal eder.
	// Yeni bağlantı üretildiğinde çağrılır: aynı anda birden fazla canlı
	// bağlantı, tahmin yüzeyini TTL boyunca çoğaltır.
	InvalidateActive(ctx context.Context, email string, at time.Time) error

	// DeleteByUser, kalıcı silmede kullanıcının bağlantılarını düşürür.
	DeleteByUser(ctx context.Context, userID string) error
}
