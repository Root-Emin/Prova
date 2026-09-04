package service

import (
	"context"
	"testing"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
)

// fakeClock, devre kesicinin soğuma süresini beklemeden ilerletir.
type fakeClock struct{ at time.Time }

func (c *fakeClock) now() time.Time          { return c.at }
func (c *fakeClock) advance(d time.Duration) { c.at = c.at.Add(d) }

func newTestRouter(clock *fakeClock) *RuleRouter {
	return NewRuleRouter(RouterConfig{
		LongInputThreshold: 100,
		CircuitThreshold:   3,
		CircuitCooldown:    30 * time.Second,
		Now:                clock.now,
	})
}

func route(t *testing.T, r *RuleRouter, in RouteInput) (model.Tier, model.RoutingRule) {
	t.Helper()
	return r.Route(context.Background(), in)
}

// Kurallar sırayla değerlendirilir ve ilk eşleşen kazanır.
func TestRuleRouter_ScoringAlwaysGoesToTheStrongTier(t *testing.T) {
	r := newTestRouter(&fakeClock{at: time.Now()})

	tier, rule := route(t, r, RouteInput{Purpose: PurposeScoring, InputLength: 10})

	if tier != model.TierStrong || rule != model.RuleScoring {
		t.Fatalf("puanlama güçlü kademeye gitmeli: %s / %s", tier, rule)
	}
}

// Puanlama kuralı, zorunlu sinyalden önce gelmeli: ikisi de güçlüye
// yönlendirir ama kayıttaki kural adı hangi kararın verildiğini anlatır.
func TestRuleRouter_ScoringWinsOverMandatorySignal(t *testing.T) {
	r := newTestRouter(&fakeClock{at: time.Now()})

	_, rule := route(t, r, RouteInput{Purpose: PurposeScoring, MandatorySignal: true})

	if rule != model.RuleScoring {
		t.Fatalf("puanlama kuralı önce gelmeli, %s geldi", rule)
	}
}

// Kararın sonucu doğrudan KALDI olabilir; o kararı hızlı modele bırakmak,
// ürünün en ağır sonucunu en zayıf modele emanet etmek olurdu.
func TestRuleRouter_MandatorySignalGoesToTheStrongTier(t *testing.T) {
	r := newTestRouter(&fakeClock{at: time.Now()})

	tier, rule := route(t, r, RouteInput{Purpose: PurposeTurn, MandatorySignal: true, InputLength: 10})

	if tier != model.TierStrong || rule != model.RuleMandatorySignal {
		t.Fatalf("zorunlu kriter sinyali güçlü kademeye gitmeli: %s / %s", tier, rule)
	}
}

func TestRuleRouter_LongInputGoesToTheStrongTier(t *testing.T) {
	r := newTestRouter(&fakeClock{at: time.Now()})

	tier, rule := route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 500})

	if tier != model.TierStrong || rule != model.RuleLongInput {
		t.Fatalf("uzun girdi güçlü kademeye gitmeli: %s / %s", tier, rule)
	}
}

// Tasarrufun tamamı bu yoldan geliyor: olağan konuşma sırası hızlı kademede.
func TestRuleRouter_OrdinaryTurnGoesToTheFastTier(t *testing.T) {
	r := newTestRouter(&fakeClock{at: time.Now()})

	tier, rule := route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 50})

	if tier != model.TierFast || rule != model.RuleDefault {
		t.Fatalf("olağan sıra hızlı kademeye gitmeli: %s / %s", tier, rule)
	}
}

// Devre eşiği aşılınca açılmalı ve çağrılar güçlüye kaymalı.
func TestRuleRouter_OpensTheCircuitAfterConsecutiveFailures(t *testing.T) {
	clock := &fakeClock{at: time.Now()}
	r := newTestRouter(clock)

	for i := 0; i < 3; i++ {
		r.ReportResult(model.TierFast, false)
	}

	if !r.IsOpen(model.TierFast) {
		t.Fatal("eşik aşıldıktan sonra devre açık olmalı")
	}
	tier, rule := route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 10})
	if tier != model.TierStrong || rule != model.RuleFailover {
		t.Fatalf("devre açıkken failover beklenir: %s / %s", tier, rule)
	}
}

// Araya giren bir başarı sayacı sıfırlamalı: ardışık olmayan hatalar
// devreyi açmamalı, yoksa günde birkaç kez tökezleyen bir sağlayıcı sürekli
// devre dışı kalır.
func TestRuleRouter_SuccessResetsTheFailureCounter(t *testing.T) {
	clock := &fakeClock{at: time.Now()}
	r := newTestRouter(clock)

	r.ReportResult(model.TierFast, false)
	r.ReportResult(model.TierFast, false)
	r.ReportResult(model.TierFast, true)
	r.ReportResult(model.TierFast, false)
	r.ReportResult(model.TierFast, false)

	if r.IsOpen(model.TierFast) {
		t.Fatal("araya giren başarı sayacı sıfırlamalı")
	}
}

// Soğuma dolunca TEK deneme yapılmalı: tüm trafiği bir anda hâlâ sağlıksız
// olabilecek bir sağlayıcıya boşaltmak, devre kesicinin çözdüğü sorunu geri
// getirir.
func TestRuleRouter_AllowsExactlyOneProbeAfterCooldown(t *testing.T) {
	clock := &fakeClock{at: time.Now()}
	r := newTestRouter(clock)

	for i := 0; i < 3; i++ {
		r.ReportResult(model.TierFast, false)
	}
	clock.advance(31 * time.Second)

	// İlk çağrı denemeye izin alır.
	tier, _ := route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 10})
	if tier != model.TierFast {
		t.Fatalf("soğuma sonrası tek deneme hızlı kademeye gitmeli, %s geldi", tier)
	}

	// Deneme sonuçlanmadan gelen ikinci çağrı yine güçlüye gitmeli.
	tier, rule := route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 10})
	if tier != model.TierStrong || rule != model.RuleFailover {
		t.Fatalf("yarı açıkken ikinci çağrı failover olmalı: %s / %s", tier, rule)
	}
}

// Deneme başarılıysa devre kapanır ve trafik geri döner.
func TestRuleRouter_ClosesTheCircuitAfterASuccessfulProbe(t *testing.T) {
	clock := &fakeClock{at: time.Now()}
	r := newTestRouter(clock)

	for i := 0; i < 3; i++ {
		r.ReportResult(model.TierFast, false)
	}
	clock.advance(31 * time.Second)
	route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 10}) // denemeyi al
	r.ReportResult(model.TierFast, true)

	if r.IsOpen(model.TierFast) {
		t.Fatal("başarılı denemeden sonra devre kapanmalı")
	}
	tier, rule := route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 10})
	if tier != model.TierFast || rule != model.RuleDefault {
		t.Fatalf("devre kapandıktan sonra trafik hızlıya dönmeli: %s / %s", tier, rule)
	}
}

// Deneme başarısızsa devre yeniden açılır ve soğuma baştan başlar.
func TestRuleRouter_ReopensTheCircuitAfterAFailedProbe(t *testing.T) {
	clock := &fakeClock{at: time.Now()}
	r := newTestRouter(clock)

	for i := 0; i < 3; i++ {
		r.ReportResult(model.TierFast, false)
	}
	clock.advance(31 * time.Second)
	route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 10})
	r.ReportResult(model.TierFast, false)

	if !r.IsOpen(model.TierFast) {
		t.Fatal("başarısız denemeden sonra devre yeniden açılmalı")
	}
	clock.advance(29 * time.Second)
	if !r.IsOpen(model.TierFast) {
		t.Fatal("soğuma süresi baştan başlamalı")
	}
}

// Kademelerin devreleri ayrı: hızlı sağlayıcının çökmesi güçlü kademeyi
// devre dışı bırakmamalı, aksi hâlde failover gidecek yer bulamaz.
func TestRuleRouter_KeepsTierCircuitsIndependent(t *testing.T) {
	clock := &fakeClock{at: time.Now()}
	r := newTestRouter(clock)

	for i := 0; i < 3; i++ {
		r.ReportResult(model.TierFast, false)
	}

	if r.IsOpen(model.TierStrong) {
		t.Fatal("hızlı kademenin hatası güçlü kademenin devresini açmamalı")
	}
	tier, _ := route(t, r, RouteInput{Purpose: PurposeScoring})
	if tier != model.TierStrong {
		t.Fatal("puanlama hâlâ güçlü kademeye gitmeli")
	}
}

// Devre açıkken uzun girdi kuralı devreye girmemeli: ikisi de güçlüye
// yönlendirir ama kayıttaki kural, kararın NEDENİNİ anlatır ve failover'ın
// gerçekleştiğinin tek kanıtıdır.
func TestRuleRouter_FailoverRuleWinsOverLongInput(t *testing.T) {
	clock := &fakeClock{at: time.Now()}
	r := newTestRouter(clock)

	for i := 0; i < 3; i++ {
		r.ReportResult(model.TierFast, false)
	}

	_, rule := route(t, r, RouteInput{Purpose: PurposeTurn, InputLength: 5000})

	if rule != model.RuleFailover {
		t.Fatalf("devre açıkken kural failover olmalı, %s geldi", rule)
	}
}
