package service

import (
	"sync"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// breakerState, devre kesicinin üç durumu.
type breakerState int

const (
	// breakerClosed, olağan durum: çağrılar geçer.
	breakerClosed breakerState = iota
	// breakerOpen, kademe devre dışı: çağrılar hiç denenmez.
	breakerOpen
	// breakerHalfOpen, soğuma süresi doldu: tek bir deneme yapılır.
	//
	// Yarı açık durum olmadan devre kesici ya sonsuza kadar açık kalır ya da
	// soğuma bitince tüm trafiği bir anda hâlâ sağlıksız olabilecek bir
	// sağlayıcıya boşaltır. Tek deneme, ikisinin arasındaki tek güvenli yol.
	breakerHalfOpen
)

const (
	defaultCircuitThreshold = 3
	defaultCircuitCooldown  = 30 * time.Second
)

// breaker, tek bir kademenin devre kesicisi.
type breaker struct {
	mu       sync.Mutex
	state    breakerState
	failures int
	openedAt time.Time

	threshold int
	cooldown  time.Duration
	now       func() time.Time
}

func newBreaker(threshold int, cooldown time.Duration, now func() time.Time) *breaker {
	if threshold <= 0 {
		threshold = defaultCircuitThreshold
	}
	if cooldown <= 0 {
		cooldown = defaultCircuitCooldown
	}
	return &breaker{threshold: threshold, cooldown: cooldown, now: now}
}

// allow, çağrının denenip denenmeyeceğini söyler.
//
// Yan etkisi var: soğuma süresi dolmuşsa durumu yarı açığa geçirir ve tek
// denemeye izin verir. Bu yüzden salt okuma gibi görünse de kilit altında.
func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case breakerClosed:
		return true
	case breakerHalfOpen:
		// Yarı açıkken yalnızca bir deneme uçuşta olmalı; ikinci çağrı
		// sonucu beklemeden geçerse deneme "tek" olmaktan çıkar.
		return false
	default:
		if b.now().Sub(b.openedAt) < b.cooldown {
			return false
		}
		b.state = breakerHalfOpen
		return true
	}
}

// isOpen, kademenin şu an kullanılamaz olduğunu söyler.
func (b *breaker) isOpen() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state == breakerOpen && b.now().Sub(b.openedAt) < b.cooldown
}

// report, çağrının sonucunu işler.
func (b *breaker) report(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if success {
		// Tek bir başarı devreyi kapatır. Kademeli iyileşme sayacı tutmak
		// daha zarif olurdu ama sağlayıcı ya cevap veriyordur ya vermiyordur;
		// aradaki gri alan pratikte gecikme olarak görünür, hata olarak değil.
		b.state = breakerClosed
		b.failures = 0
		return
	}

	b.failures++
	if b.state == breakerHalfOpen || b.failures >= b.threshold {
		// Yarı açıkken gelen tek hata devreyi yeniden açar: deneme tam da
		// bunu ölçmek içindi.
		b.state = breakerOpen
		b.openedAt = b.now()
	}
}

// breakers, kademe başına devre kesici.
type breakers struct {
	fast   *breaker
	strong *breaker
}

func newBreakers(threshold int, cooldown time.Duration, now func() time.Time) *breakers {
	return &breakers{
		fast:   newBreaker(threshold, cooldown, now),
		strong: newBreaker(threshold, cooldown, now),
	}
}

func (b *breakers) forTier(tier model.Tier) *breaker {
	if tier == model.TierStrong {
		return b.strong
	}
	return b.fast
}
