package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
)

// DeviceChallengeRepository, cihaz imza meydan okumalarını saklar.
type DeviceChallengeRepository interface {
	Create(ctx context.Context, challenge *model.DeviceChallenge) error

	// GetLatestUsable, adres ve cihaz için henüz kullanılmamış son
	// challenge'ı döndürür.
	GetLatestUsable(ctx context.Context, email, fingerprintHash string, now time.Time) (*model.DeviceChallenge, error)

	// Consume, challenge'ı yakar.
	//
	// Koşul sorgunun içinde: challenge tek kullanımlık olmazsa yakalanan bir
	// imza, o challenge'ın ömrü boyunca yeniden oynatılabilir.
	Consume(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)
}
