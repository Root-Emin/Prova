// Package realtime, oturum olaylarını abonelere dağıtır.
package realtime

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
	provaUC "github.com/masterfabric-go/masterfabric/internal/application/prova/usecase"
)

// subscriberBuffer, abone başına tampon boyutu.
//
// Tamponlu kanal kullanılıyor çünkü yayın yolu asla bloklanmamalı: yavaş bir
// WebSocket istemcisi, o oturumun konuşma sırasını işleyen goroutine'i
// bekletirse görüşmenin tamamı durur. Tampon dolduğunda olay düşürülür ve
// loglanır — geç bir olay, kilitlenmiş bir oturumdan iyidir.
const subscriberBuffer = 32

// SessionBroker, oturum olaylarının süreç içi dağıtıcısı.
//
// Süreç içi olması bilinçli: Prova tek süreçte çalışıyor ve abonelikler
// oturum sahibine bağlı. Çok örnekli bir dağıtıma geçildiğinde buranın
// Redis pub/sub ile değiştirilmesi gerekir; arayüz aynı kalır.
type SessionBroker struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[int]chan provaUC.SessionEvent
	nextID      int
	log         *slog.Logger
}

// NewSessionBroker wires the broker.
func NewSessionBroker(log *slog.Logger) *SessionBroker {
	return &SessionBroker{
		subscribers: map[uuid.UUID]map[int]chan provaUC.SessionEvent{},
		log:         log,
	}
}

// Publish implements usecase.SessionPublisher.
func (b *SessionBroker) Publish(sessionID uuid.UUID, event provaUC.SessionEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers[sessionID] {
		select {
		case ch <- event:
		default:
			b.log.Warn("oturum olayı düşürüldü: abone tamponu dolu",
				"session_id", sessionID, "event", event.Type)
		}
	}
}

// Subscribe, bir oturumun olay akışını açar.
//
// Dönen fonksiyon aboneliği kapatır ve kanalı sonlandırır. Çağrılmazsa abone
// süresiz yaşar; bu yüzden resolver onu defer ile bağlar.
func (b *SessionBroker) Subscribe(sessionID uuid.UUID) (<-chan provaUC.SessionEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers[sessionID] == nil {
		b.subscribers[sessionID] = map[int]chan provaUC.SessionEvent{}
	}
	b.nextID++
	id := b.nextID
	ch := make(chan provaUC.SessionEvent, subscriberBuffer)
	b.subscribers[sessionID][id] = ch

	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subs, ok := b.subscribers[sessionID]
		if !ok {
			return
		}
		if existing, ok := subs[id]; ok {
			delete(subs, id)
			close(existing)
		}
		if len(subs) == 0 {
			delete(b.subscribers, sessionID)
		}
	}
}

// SubscriberCount, testler ve sağlık çıktısı için abone sayısı.
func (b *SessionBroker) SubscriberCount(sessionID uuid.UUID) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[sessionID])
}

var _ provaUC.SessionPublisher = (*SessionBroker)(nil)
