package model

import (
	"time"

	"github.com/google/uuid"
)

// SessionStatus, oturumun yaşam evresi.
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusScoring   SessionStatus = "scoring"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusAbandoned SessionStatus = "abandoned"
)

// Session, bir çalışanın bir senaryoyu oynadığı kayıt.
//
// Oturum sürümlenebilir bir belge değildir — oynanır ve biter. Ama
// sürümlenebilir belgelere sürüm referansıyla bağlanır: senaryo, karakter ve
// rubrik sürümleri oturum başlarken buraya yazılır ve bir daha değişmez.
// Sertifikasyon iddiası tam olarak buna dayanıyor; içerik sonradan
// güncellense bile oturum kendi sürümüyle puanlanır.
type Session struct {
	ID    uuid.UUID `bson:"_id" json:"id"`
	OrgID uuid.UUID `bson:"org_id" json:"org_id"`
	// EmployeeID, oturumu oynayan çalışan. Kalıcı silmede
	// kimliksizleştirilir; oturum ve puanlar kurum istatistiği için kalır.
	EmployeeID uuid.UUID `bson:"employee_id" json:"employee_id"`
	// Anonymized, çalışan kimliğinin silinip silinmediği. Ayrı bir bayrak
	// tutuluyor çünkü kimliksizleştirilmiş bir oturumu "kimliği hiç
	// olmayan" bir oturumdan ayırt edebilmek gerekir.
	Anonymized bool `bson:"anonymized" json:"anonymized"`
	// DeviceID, oturumun oynandığı kayıtlı cihaz. Sertifika taşıyan bir sınav
	// bilinen bir makineye bağlanabilmeli.
	DeviceID *uuid.UUID `bson:"device_id,omitempty" json:"device_id,omitempty"`

	Status SessionStatus `bson:"status" json:"status"`

	// Oturum başlarken dondurulan sürümler.
	Scenario  Reference `bson:"scenario" json:"scenario"`
	Character Reference `bson:"character" json:"character"`
	Rubric    Reference `bson:"rubric" json:"rubric"`

	// MaxTurns, oturum başlarken senaryodan kopyalanır. Senaryo sonradan
	// değişse bile bu oturumun limiti değişmez.
	MaxTurns  int `bson:"max_turns" json:"max_turns"`
	TurnCount int `bson:"turn_count" json:"turn_count"`

	StartedAt time.Time  `bson:"started_at" json:"started_at"`
	EndedAt   *time.Time `bson:"ended_at,omitempty" json:"ended_at,omitempty"`
}

// IsActive reports whether more turns may be submitted.
func (s *Session) IsActive() bool { return s.Status == SessionStatusActive }

// HasTurnsLeft reports whether the turn budget still allows a submission.
func (s *Session) HasTurnsLeft() bool { return s.TurnCount < s.MaxTurns }

// TurnRole, konuşma sırasının kimden geldiği.
type TurnRole string

const (
	TurnRoleEmployee  TurnRole = "employee"
	TurnRoleCharacter TurnRole = "character"
)

// Turn, tek bir konuşma sırası.
type Turn struct {
	ID        uuid.UUID `bson:"_id" json:"id"`
	OrgID     uuid.UUID `bson:"org_id" json:"org_id"`
	SessionID uuid.UUID `bson:"session_id" json:"session_id"`
	// RequestID, bir SubmitTurn denemesini iki kez uygulamayı önleyen
	// istemci-idempotency anahtarıdır. Eski turn'lerde boş kalabilir.
	RequestID *uuid.UUID `bson:"request_id,omitempty" json:"-"`
	// Index, oturum içindeki sıra numarası. Puanlama modeli alıntıyı bu
	// numarayla işaretler ve alıntı doğrulaması bu numaradaki metinde arar.
	Index int      `bson:"index" json:"index"`
	Role  TurnRole `bson:"role" json:"role"`
	// Text, transkriptin kendisi: maskelenmemiş orijinal metin.
	//
	// Maskeleme yalnızca LLM'e giden kopyada yapılır. Transkriptte de
	// maskelenseydi puanlama, çalışanın gerçekten ne söylediğini göremezdi ve
	// "kişisel veri ifşa etti mi" sorusu cevaplanamaz hâle gelirdi.
	Text string `bson:"text" json:"text"`
	// Signals, bu sırada tetiklenen rubrik kriter anahtarları.
	Signals []string `bson:"signals" json:"signals"`
	// MaskedFieldCount, LLM'e giden kopyada maskelenen kişisel veri alanı
	// sayısı. KVKK hesap verebilirliği için kaydediliyor.
	MaskedFieldCount int       `bson:"masked_field_count" json:"masked_field_count"`
	CreatedAt        time.Time `bson:"created_at" json:"created_at"`
}

// AnonymizedTurnText, kimliksizleştirmede çalışanın metninin yerine yazılan
// işaret.
//
// Boş metin yerine açık bir işaret kullanılıyor: transkriptte boşluk gören
// biri veri kaybı mı yoksa silme mi olduğunu ayırt edemezdi.
const AnonymizedTurnText = "[silindi]"
