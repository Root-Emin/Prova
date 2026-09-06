package usecase

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	notify "github.com/masterfabric-go/masterfabric/internal/domain/notification/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/notification/template"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/ratelimit"
)

// RequestLoginCodeUseCase issues a one-time sign-in code and mails it.
//
// The single hardest requirement on this path is that its observable behaviour
// must not depend on whether the address belongs to an account: same status,
// same body, same duration. Three separate mechanisms enforce that — an
// identical response value, a response-time floor, and a rate limit keyed on
// values the caller already knows.
type RequestLoginCodeUseCase struct {
	RequestDeps

	sleep func(time.Duration)
	now   func() time.Time
}

// RequestDeps, kod isteme akışının bağımlılıkları.
type RequestDeps struct {
	Users   repository.UserRepository
	Codes   repository.LoginCodeRepository
	CodeSvc service.LoginCodeService
	Sender  notify.Sender
	Limiter ratelimit.Limiter
	Audit   auditService.Recorder
	Cfg     config.AuthConfig
	Log     *slog.Logger

	// MagicLinks, aynı e-postaya konacak tek kullanımlık bağlantıyı üretir.
	// Bağlı değilse ileti yalnızca kodu taşır; giriş yine çalışır.
	MagicLinks MagicLinkIssuer
	// WebBaseURL, bağlantının işaret ettiği web arayüzünün kökü.
	WebBaseURL string
}

// MagicLinkIssuer, e-postaya konacak tek kullanımlık bağlantı token'ını üretir.
//
// Arayüz burada, uygulaması Faz 6'da. Kod ve bağlantı aynı iletide gider, bu
// yüzden üretimi bu akışın içinde olmak zorunda.
type MagicLinkIssuer interface {
	Issue(ctx context.Context, email string, now time.Time) (token string, err error)
}

// NewRequestLoginCodeUseCase wires the use case.
func NewRequestLoginCodeUseCase(deps RequestDeps) *RequestLoginCodeUseCase {
	if deps.Audit == nil {
		deps.Audit = auditService.NoopRecorder{}
	}
	return &RequestLoginCodeUseCase{
		RequestDeps: deps,
		sleep:       time.Sleep,
		now:         time.Now,
	}
}

// Execute issues a code for the address. requestIP is used for rate limiting
// only and may be empty.
func (uc *RequestLoginCodeUseCase) Execute(ctx context.Context, req dto.RequestLoginCodeRequest, requestIP string) (*dto.RequestLoginCodeResponse, error) {
	started := uc.now()
	defer uc.padResponseTime(started)

	email := model.NormalizeEmail(req.Email)
	if email == "" {
		return nil, domainErr.New(domainErr.ErrValidation, "email is required", nil)
	}

	// IP first: it is the only limit that constrains an attacker sweeping many
	// addresses, and it must apply before any work that touches the database.
	if err := uc.checkLimit(ctx, "ip:"+requestIP, uc.Cfg.MaxRequestsPerIP, requestIP != ""); err != nil {
		return nil, err
	}
	if err := uc.checkLimit(ctx, "email:"+email, uc.Cfg.MaxRequestsPerEmail, true); err != nil {
		return nil, err
	}

	if err := uc.checkCooldown(ctx, email); err != nil {
		return nil, err
	}

	user, err := uc.Users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		// Company invites leave users inactive/unverified until first launch.
		// Those provisioned accounts must still receive a login code so the
		// desktop/web verify step can activate them. Deleted and suspended
		// accounts stay silent so this endpoint cannot probe admin state.
		// Keep the response generic either way (anti-enumeration).
		if user.IsDeleted() || user.Status == model.UserStatusSuspended {
			return uc.response(), nil
		}
	case errors.Is(err, domainErr.ErrNotFound):
		// Unknown addresses never receive mail and never provision an account.
		return uc.response(), nil
	default:
		return nil, err
	}

	code, digest, err := generateLoginCode(uc.CodeSvc, user, email)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate login code", err)
	}

	now := uc.now().UTC()

	// Retire outstanding codes before issuing a new one. Several live codes for
	// one address multiply the guessing surface for the whole TTL, and a user
	// who requested a second code has already stopped watching for the first.
	if err := uc.Codes.InvalidateActive(ctx, email, model.LoginCodePurposeLogin, now); err != nil {
		return nil, err
	}

	record := &model.LoginCode{
		Email:       email,
		CodeDigest:  digest,
		Purpose:     model.LoginCodePurposeLogin,
		MaxAttempts: uc.Cfg.MaxAttempts,
		ExpiresAt:   now.Add(uc.Cfg.CodeTTL),
		RequestIP:   requestIP,
		CreatedAt:   now,
	}
	if user != nil {
		record.UserID = user.ID
	}
	if err := uc.Codes.Create(ctx, record); err != nil {
		return nil, err
	}

	// Aynı ileti hem kodu hem bağlantıyı taşır: kullanıcı hangisini isterse
	// onu kullanır. Bağlantı üretimi başarısız olursa ileti kodla gider —
	// bağlantı bir kolaylıktır, girişin tek yolu değil.
	// A first-launch invite must be redeemed by Desktop so the mailbox proof
	// and device pairing happen as one operation. Magic links remain a
	// convenience only for accounts that have already paired once.
	magicToken := ""
	if user.IsEmailVerified() {
		magicToken = uc.issueMagicLink(ctx, email, now)
	}
	msg := template.LoginCode(notifyModel.Address{Email: email}, code, uc.Cfg.CodeTTL, uc.magicLinkURL(magicToken))
	messageID, err := uc.Sender.Send(ctx, msg)
	if err != nil {
		// A code nobody can read is the same as no code at all, so this is a
		// hard failure rather than a silent success. The stored code is left in
		// place: it expires on its own, and burning it here would let a
		// provider hiccup lock the address out of its own retry.
		uc.Log.ErrorContext(ctx, "login code delivery failed",
			"provider", uc.Sender.Name(),
			"error", err,
		)
		return nil, domainErr.New(domainErr.ErrInternal, "failed to send login code", err)
	}

	// The address is deliberately absent from this line: it is the one field
	// that would turn the log stream into the account list the endpoint refuses
	// to be.
	uc.Log.InfoContext(ctx, "login code sent",
		"provider", uc.Sender.Name(),
		"message_id", messageID,
		"code_id", record.ID,
		"magic_link", magicToken != "",
	)

	// Kimlik henüz doğrulanmadığı için bu kayıtta organizasyon yoktur; sıfır
	// UUID "kiracı öncesi olay" anlamına gelir. Adres de yazılmaz: denetim
	// kaydı hiç silinmez, ve oraya yazılan her adres kalıcı bir hesap listesi
	// üretirdi. Kayıt, kod kimliği üzerinden doğrulama olayına bağlanır.
	uc.Audit.Record(ctx, auditService.Entry{
		Action:       auditService.ActionLoginRequested,
		ResourceType: "login_code",
		ResourceID:   record.ID.String(),
		IPAddress:    requestIP,
		Metadata:     map[string]any{"magic_link": magicToken != ""},
	})
	if magicToken != "" {
		uc.Audit.Record(ctx, auditService.Entry{
			Action:       auditService.ActionMagicLinkSent,
			ResourceType: "login_code",
			ResourceID:   record.ID.String(),
			IPAddress:    requestIP,
		})
	}

	return uc.response(), nil
}

// issueMagicLink, tek kullanımlık bağlantı token'ını üretir. Üretici bağlı
// değilse boş dönerek iletinin yalnızca kodla gitmesini sağlar.
func (uc *RequestLoginCodeUseCase) issueMagicLink(ctx context.Context, email string, now time.Time) string {
	if uc.MagicLinks == nil || uc.WebBaseURL == "" {
		return ""
	}
	token, err := uc.MagicLinks.Issue(ctx, email, now)
	if err != nil {
		uc.Log.ErrorContext(ctx, "magic link üretilemedi, ileti yalnızca kodla gidiyor", "error", err)
		return ""
	}
	return token
}

// magicLinkURL, token'ı web arayüzünün doğrulama sayfasına bağlar.
//
// Bağlantı tarayıcıda açılır; Electron deep-link'i bilerek kullanılmaz. Bir
// deep-link, e-postadaki bağlantıyı yerel bir uygulamanın kayıtlı şemasına
// teslim eder ve o şemayı kaydeden her uygulama token'ı görebilir.
func (uc *RequestLoginCodeUseCase) magicLinkURL(token string) string {
	if token == "" {
		return ""
	}
	return strings.TrimRight(uc.WebBaseURL, "/") + "/auth/magic?token=" + url.QueryEscape(token)
}

func (uc *RequestLoginCodeUseCase) response() *dto.RequestLoginCodeResponse {
	return &dto.RequestLoginCodeResponse{
		Sent:               true,
		ExpiresInSeconds:   int(uc.Cfg.CodeTTL.Seconds()),
		ResendAfterSeconds: int(uc.Cfg.ResendCooldown.Seconds()),
	}
}

func (uc *RequestLoginCodeUseCase) checkLimit(ctx context.Context, key string, limit int, enabled bool) error {
	if !enabled || limit <= 0 || uc.Limiter == nil {
		return nil
	}
	allowed, retryAfter, err := uc.Limiter.Allow(ctx, key, limit, uc.Cfg.RateLimitWindow)
	if err != nil {
		// Failing open is the deliberate choice: a Redis outage must not take
		// the only sign-in path down with it. The per-address cooldown below
		// runs off PostgreSQL and still throttles the obvious abuse.
		uc.Log.WarnContext(ctx, "rate limiter unavailable, allowing request", "error", err)
		return nil
	}
	if !allowed {
		return domainErr.New(domainErr.ErrRateLimited,
			"too many login code requests, retry in "+retryAfter.Round(time.Second).String(), nil)
	}
	return nil
}

// checkCooldown enforces the minimum gap between two codes for one address.
func (uc *RequestLoginCodeUseCase) checkCooldown(ctx context.Context, email string) error {
	if uc.Cfg.ResendCooldown <= 0 {
		return nil
	}

	latest, err := uc.Codes.GetLatest(ctx, email, model.LoginCodePurposeLogin)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil
		}
		return err
	}

	elapsed := uc.now().UTC().Sub(latest.CreatedAt)
	if elapsed < uc.Cfg.ResendCooldown {
		wait := (uc.Cfg.ResendCooldown - elapsed).Round(time.Second)
		return domainErr.New(domainErr.ErrRateLimited,
			"a code was already sent, retry in "+wait.String(), nil)
	}
	return nil
}

// padResponseTime holds the response until the configured floor has elapsed.
//
// Wording alone does not hide account existence. A known address costs a user
// lookup, a code write and a provider round trip; an unknown one costs a
// lookup. That difference is measurable over enough samples, so every answer is
// stretched to the same floor.
func (uc *RequestLoginCodeUseCase) padResponseTime(started time.Time) {
	if uc.Cfg.MinResponseTime <= 0 {
		return
	}
	if remaining := uc.Cfg.MinResponseTime - uc.now().Sub(started); remaining > 0 {
		uc.sleep(remaining)
	}
}
