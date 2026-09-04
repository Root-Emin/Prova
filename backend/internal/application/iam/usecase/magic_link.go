package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	auditService "github.com/masterfabric-go/masterfabric/internal/domain/audit/service"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// MagicLinkTokens, token üretimi ve hash'leme.
type MagicLinkTokens interface {
	Generate() (token, hash string, err error)
	Hash(token string) string
}

// MagicLinkIssuerImpl, kod isteme akışına bağlanan bağlantı üreticisi.
//
// Kodla aynı e-postada gider: kullanıcı hangisini isterse onu kullanır.
// İkisini ayrı iletilere bölmek, kullanıcıya hangi iletinin hangi girişe ait
// olduğunu çözme yükü bindirirdi.
type MagicLinkIssuerImpl struct {
	links  repository.MagicLinkRepository
	users  repository.UserRepository
	tokens MagicLinkTokens
	ttl    time.Duration
	log    *slog.Logger
}

// NewMagicLinkIssuer wires the issuer.
func NewMagicLinkIssuer(
	links repository.MagicLinkRepository,
	users repository.UserRepository,
	tokens MagicLinkTokens,
	ttl time.Duration,
	log *slog.Logger,
) *MagicLinkIssuerImpl {
	return &MagicLinkIssuerImpl{links: links, users: users, tokens: tokens, ttl: ttl, log: log}
}

// Issue implements RequestLoginCodeUseCase's MagicLinkIssuer.
func (i *MagicLinkIssuerImpl) Issue(ctx context.Context, email string, now time.Time) (string, error) {
	token, hash, err := i.tokens.Generate()
	if err != nil {
		return "", err
	}

	// Eski bağlantılar önce iptal ediliyor. Aynı adres için birden fazla
	// canlı bağlantı, tahmin yüzeyini TTL boyunca çoğaltır — ve ikinci
	// bağlantıyı isteyen kullanıcı zaten birincisini beklemeyi bırakmıştır.
	if err := i.links.InvalidateActive(ctx, email, now); err != nil {
		return "", err
	}

	link := &model.MagicLink{
		Email:     email,
		TokenHash: hash,
		ExpiresAt: now.Add(i.ttl),
		CreatedAt: now,
	}
	// Kullanıcı varsa bağlanır; yoksa bağlantı yalnızca adrese bağlı kalır ve
	// hesap doğrulama anında açılır. Kod hattıyla aynı davranış: doğrulanmamış
	// bir adres için kullanıcı satırı oluşturulmaz.
	if user, err := i.users.GetByEmail(ctx, email); err == nil {
		link.UserID = &user.ID
	} else if !errors.Is(err, domainErr.ErrNotFound) {
		return "", err
	}

	if err := i.links.Create(ctx, link); err != nil {
		return "", err
	}
	return token, nil
}

// VerifyMagicLinkUseCase, bağlantıyı oturuma çevirir.
//
// Kod doğrulamasıyla aynı kuyruğu kullanır (VerifyLoginCodeUseCase.CompleteLogin):
// cihaz eşleştirme, organizasyon çözümü ve denetim kaydı iki giriş yolunda da
// aynı olmalı, yoksa zamanla farklı güvenlik davranışına kayarlar.
type VerifyMagicLinkUseCase struct {
	links      repository.MagicLinkRepository
	users      repository.UserRepository
	tokens     MagicLinkTokens
	login      *VerifyLoginCodeUseCase
	audit      auditService.Recorder
	selfSignup bool
	log        *slog.Logger
	now        func() time.Time
}

// NewVerifyMagicLinkUseCase wires the use case.
func NewVerifyMagicLinkUseCase(
	links repository.MagicLinkRepository,
	users repository.UserRepository,
	tokens MagicLinkTokens,
	login *VerifyLoginCodeUseCase,
	audit auditService.Recorder,
	selfSignup bool,
	log *slog.Logger,
) *VerifyMagicLinkUseCase {
	if audit == nil {
		audit = auditService.NoopRecorder{}
	}
	return &VerifyMagicLinkUseCase{
		links: links, users: users, tokens: tokens, login: login,
		audit: audit, selfSignup: selfSignup, log: log, now: time.Now,
	}
}

// errInvalidLink, her reddin toplandığı tek cevap.
//
// Kod hattındaki mantığın aynısı: süresi dolmuş, kullanılmış, iptal edilmiş
// ve hiç var olmayan bağlantı aynı hatayı döndürür. Ayırt etmek, saldırgana
// hangi varsayımının yanlış olduğunu söylerdi ve meşru kullanıcının bir
// sonraki adımı dört durumda da aynı: yeni bağlantı iste.
func errInvalidLink() error {
	return domainErr.New(domainErr.ErrUnauthorized, "geçersiz veya süresi dolmuş bağlantı", nil)
}

// Execute redeems the link for a session.
func (uc *VerifyMagicLinkUseCase) Execute(ctx context.Context, req dto.VerifyMagicLinkRequest, requestIP string) (*dto.VerifyLoginCodeResponse, error) {
	if req.Token == "" {
		return nil, errInvalidLink()
	}
	now := uc.now().UTC()

	link, err := uc.links.GetByTokenHash(ctx, uc.tokens.Hash(req.Token))
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, errInvalidLink()
		}
		return nil, err
	}
	if !link.IsRedeemable(now) {
		return nil, errInvalidLink()
	}

	// Bağlantı her şeyden önce yakılıyor: aynı bağlantıyla gelen iki istek
	// iki oturum üretmemeli, ve yakma koşulu sorgunun içinde olduğu için
	// yarış kaybeden istek burada durur.
	consumed, err := uc.links.Consume(ctx, link.ID.String(), now)
	if err != nil {
		return nil, err
	}
	if !consumed {
		return nil, errInvalidLink()
	}

	user, err := uc.resolveUser(ctx, link.Email, now)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, domainErr.New(domainErr.ErrForbidden, "hesap etkin değil", nil)
	}

	userID := user.ID
	uc.audit.Record(ctx, auditService.Entry{
		UserID:       &userID,
		Action:       auditService.ActionMagicLinkUsed,
		ResourceType: "magic_link",
		ResourceID:   link.ID.String(),
		IPAddress:    requestIP,
	})

	return uc.login.CompleteLogin(ctx, user, req.Device, now, "magic_link")
}

// resolveUser, adresin hesabını döndürür, gerekiyorsa oluşturur.
func (uc *VerifyMagicLinkUseCase) resolveUser(ctx context.Context, email string, now time.Time) (*model.User, error) {
	user, err := uc.users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		if user.EmailVerifiedAt == nil {
			// Bağlantıya tıklamak da mailbox'ı kanıtlar; kodla aynı ağırlıkta.
			verifiedAt := now
			user.EmailVerifiedAt = &verifiedAt
			if err := uc.users.Update(ctx, user); err != nil {
				return nil, err
			}
		}
		return user, nil

	case errors.Is(err, domainErr.ErrNotFound):
		if !uc.selfSignup {
			return nil, errInvalidLink()
		}
		verifiedAt := now
		user = &model.User{
			Email:           model.NormalizeEmail(email),
			Status:          model.UserStatusActive,
			EmailVerifiedAt: &verifiedAt,
		}
		if err := uc.users.Create(ctx, user); err != nil {
			return nil, err
		}
		return user, nil

	default:
		return nil, err
	}
}
