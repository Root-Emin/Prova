// Package service, denetim kaydının uygulama katmanına bakan yüzünü tanımlar.
//
// Denetim kaydı HTTP seviyesinde değil, use case seviyesinde tutulur. Bunun
// nedeni tek bir cümlede özetlenebilir: "POST /graphql" hiçbir şey anlatmaz.
// KVKK hesap verebilirlik ilkesinin istediği şey hangi kullanıcının hangi
// veriyi ne zaman işlediğidir; bunu yalnızca kararı veren kod bilir.
package service

import (
	"context"

	"github.com/google/uuid"
)

// Action, denetime düşen olayın adı.
//
// Serbest metin yerine sabit kullanılır: denetim kaydı sorgulanabilir olmalı,
// ve "login.success" ile "Login Success" iki ayrı olay hâline gelirse
// sorgulanamaz.
type Action string

// Kimlik olayları.
const (
	ActionLoginRequested Action = "auth.login.requested"
	ActionLoginSucceeded Action = "auth.login.succeeded"
	ActionLoginFailed    Action = "auth.login.failed"
	ActionLoginLocked    Action = "auth.login.locked"
	ActionMagicLinkSent  Action = "auth.magic_link.sent"
	ActionMagicLinkUsed  Action = "auth.magic_link.used"
	ActionLogout         Action = "auth.logout"
	ActionRefreshRotated Action = "auth.refresh.rotated"
	ActionRefreshReuse   Action = "auth.refresh.reuse_detected"
)

// Cihaz olayları.
const (
	ActionDevicePaired            Action = "device.paired"
	ActionDeviceRevoked           Action = "device.revoked"
	ActionDeviceChallengeIssued   Action = "device.challenge.issued"
	ActionDeviceChallengeVerified Action = "device.challenge.verified"
	ActionDeviceChallengeFailed   Action = "device.challenge.failed"
)

// Hesap yaşam döngüsü olayları.
const (
	ActionProfileUpdated   Action = "account.profile.updated"
	ActionDataExported     Action = "account.data.exported"
	ActionDeletionRequeste Action = "account.deletion.requested"
	ActionDeletionCanceled Action = "account.deletion.canceled"
	ActionAccountPurged    Action = "account.purged"
)

// İçerik ve puanlama olayları.
const (
	ActionLLMProfileChanged Action = "llm.profile.changed"
	ActionLLMProfileTested  Action = "llm.profile.tested"
	ActionScoreOverridden   Action = "score.overridden"
	ActionContentPublished  Action = "content.published"
	ActionContentVersioned  Action = "content.versioned"
	ActionSessionStarted    Action = "session.started"
	ActionSessionEnded      Action = "session.ended"
)

// Entry, kaydedilecek tek bir olay.
type Entry struct {
	OrgID        uuid.UUID
	UserID       *uuid.UUID
	RequestID    string
	Action       Action
	ResourceType string
	ResourceID   string
	// Metadata, olaya özgü ek alanlar. Kişisel veri buraya yazılmaz; denetim
	// kaydı silinmeyen tek koleksiyondur, dolayısıyla içine yazılan her kişisel
	// veri kalıcıdır.
	Metadata  map[string]any
	IPAddress string
	UserAgent string
}

// Recorder, olayları kalıcılaştıran port.
//
// Record hiçbir zaman hata döndürmez: denetim yazımının başarısızlığı, denetime
// konu olan işlemi geri almaz. Kayıp kayıt loglanır, çağıran akış devam eder.
type Recorder interface {
	Record(ctx context.Context, entry Entry)
}

// NoopRecorder, denetim deposu yokken kullanılan boş uygulama. Testlerde ve
// veritabanısız boot'ta çağıranların nil kontrolü yapmasını gereksiz kılar.
type NoopRecorder struct{}

// Record implements Recorder.
func (NoopRecorder) Record(context.Context, Entry) {}
