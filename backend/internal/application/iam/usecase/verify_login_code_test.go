package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	iamEvent "github.com/masterfabric-go/masterfabric/internal/domain/iam/event"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

type verifyFixture struct {
	uc      *VerifyLoginCodeUseCase
	users   *fakeUserRepo
	codes   *fakeCodeRepo
	devices *fakeDeviceRepo
	sender  *fakeSender
	limiter *fakeLimiter
	bus     *fakeEventBus
	slept   time.Duration
	clock   time.Time
}

func newVerifyFixture(t *testing.T, cfg config.AuthConfig, users *fakeUserRepo) *verifyFixture {
	t.Helper()

	f := &verifyFixture{
		users:   users,
		codes:   newFakeCodeRepo(),
		devices: newFakeDeviceRepo(),
		sender:  newFakeSender(),
		limiter: newFakeLimiter(),
		bus:     newFakeEventBus(),
		clock:   time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
	}
	f.uc = NewVerifyLoginCodeUseCase(
		f.users, f.codes, f.devices, fixedCodeService{code: "123456"},
		fakeAuthService{}, f.sender, f.limiter, f.bus,
		cfg, config.JWTConfig{ExpirationHours: 24}, discardLogger(),
	)
	f.uc.now = func() time.Time { return f.clock }
	f.uc.sleep = func(d time.Duration) { f.slept += d }
	return f
}

// seedCode puts a live code on file for the address.
func (f *verifyFixture) seedCode(email string, cfg config.AuthConfig) *model.LoginCode {
	code := &model.LoginCode{
		Email:       model.NormalizeEmail(email),
		CodeDigest:  digestOf("123456"),
		Purpose:     model.LoginCodePurposeLogin,
		MaxAttempts: cfg.MaxAttempts,
		ExpiresAt:   f.clock.Add(cfg.CodeTTL),
		CreatedAt:   f.clock,
	}
	if err := f.codes.Create(context.Background(), code); err != nil {
		panic(err)
	}
	return f.codes.codes[len(f.codes.codes)-1]
}

func TestVerifyLoginCode_IssuesTokenForCorrectCode(t *testing.T) {
	cfg := testAuthConfig()
	user := &model.User{Email: "user@corp.com", Status: model.UserStatusActive}
	f := newVerifyFixture(t, cfg, newFakeUserRepo(user))
	f.seedCode("user@corp.com", cfg)

	resp, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "User@Corp.com", Code: "123456"}, "203.0.113.1")

	require.NoError(t, err)
	assert.Equal(t, "token-for-"+user.ID.String(), resp.Token)
	assert.Equal(t, f.clock.Add(24*time.Hour), resp.ExpiresAt)
	assert.Equal(t, "user@corp.com", resp.User.Email)
	assert.Nil(t, resp.Device, "no fingerprint was sent, so nothing may be paired")
}

// Redeeming the code proves the mailbox. That fact is recorded once.
func TestVerifyLoginCode_MarksEmailVerified(t *testing.T) {
	cfg := testAuthConfig()
	user := &model.User{Email: "user@corp.com", Status: model.UserStatusActive}
	f := newVerifyFixture(t, cfg, newFakeUserRepo(user))
	f.seedCode("user@corp.com", cfg)

	_, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456"}, "")

	require.NoError(t, err)
	require.NotNil(t, user.EmailVerifiedAt)
	assert.Equal(t, f.clock, *user.EmailVerifiedAt)
}

// The code is single-use. Two requests carrying the same correct code must not
// both produce a session.
func TestVerifyLoginCode_CodeCannotBeReplayed(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))
	f.seedCode("user@corp.com", cfg)

	_, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456"}, "")
	require.NoError(t, err)

	_, err = f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456"}, "")
	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
}

// Every rejection reads the same. Telling the caller *which* assumption was
// wrong is free information for a guesser and useless to a real user.
func TestVerifyLoginCode_AllRejectionsLookAlike(t *testing.T) {
	cfg := testAuthConfig()

	cases := []struct {
		name  string
		setup func(f *verifyFixture)
		code  string
	}{
		{
			name:  "no code on file",
			setup: func(*verifyFixture) {},
			code:  "123456",
		},
		{
			name:  "wrong code",
			setup: func(f *verifyFixture) { f.seedCode("user@corp.com", cfg) },
			code:  "999999",
		},
		{
			name: "expired code",
			setup: func(f *verifyFixture) {
				f.seedCode("user@corp.com", cfg)
				f.clock = f.clock.Add(cfg.CodeTTL + time.Minute)
			},
			code: "123456",
		},
		{
			name: "exhausted code",
			setup: func(f *verifyFixture) {
				c := f.seedCode("user@corp.com", cfg)
				c.Attempts = cfg.MaxAttempts
			},
			code: "123456",
		},
	}

	var messages []string
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
				Email: "user@corp.com", Status: model.UserStatusActive,
			}))
			tc.setup(f)

			_, err := f.uc.Execute(context.Background(),
				dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: tc.code}, "")

			require.Error(t, err)
			assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
			messages = append(messages, err.Error())
		})
	}

	for _, msg := range messages {
		assert.Equal(t, messages[0], msg, "rejection reasons must be indistinguishable")
	}
}

func TestVerifyLoginCode_WrongCodeChargesTheAttemptBudget(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))
	code := f.seedCode("user@corp.com", cfg)

	for i := 1; i <= 3; i++ {
		_, err := f.uc.Execute(context.Background(),
			dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "000000"}, "")
		require.Error(t, err)
		assert.Equal(t, i, code.Attempts)
	}
	assert.Nil(t, code.ConsumedAt, "the code stays usable while budget remains")
}

// Once the budget is gone the code is burned, so it cannot be revived by
// waiting out the rest of its TTL.
func TestVerifyLoginCode_BurnsCodeWhenAttemptsRunOut(t *testing.T) {
	cfg := testAuthConfig()
	cfg.MaxAttempts = 3
	f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))
	code := f.seedCode("user@corp.com", cfg)

	for i := 0; i < 3; i++ {
		_, err := f.uc.Execute(context.Background(),
			dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "000000"}, "")
		require.Error(t, err)
	}

	assert.NotNil(t, code.ConsumedAt, "an exhausted code must be burned")

	// Even the correct code is worthless now.
	_, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456"}, "")
	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
}

// The per-code budget cannot see an attacker who spreads guesses over many
// addresses. The IP limit is what covers that.
func TestVerifyLoginCode_EnforcesPerIPLimit(t *testing.T) {
	cfg := testAuthConfig()
	cfg.MaxRequestsPerIP = 2
	f := newVerifyFixture(t, cfg, newFakeUserRepo())

	for i := 0; i < 2; i++ {
		_, err := f.uc.Execute(context.Background(),
			dto.VerifyLoginCodeRequest{Email: "a@corp.com", Code: "000000"}, "203.0.113.7")
		require.ErrorIs(t, err, domainErr.ErrUnauthorized)
	}

	_, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "b@corp.com", Code: "000000"}, "203.0.113.7")
	assert.ErrorIs(t, err, domainErr.ErrRateLimited)
}

func TestVerifyLoginCode_PadsRejectionToFloor(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo())

	_, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "nobody@corp.com", Code: "123456"}, "")

	require.Error(t, err)
	assert.Equal(t, cfg.MinResponseTime, f.slept)
}

// Self-signup provisions the account here, not at request time: codes that are
// never redeemed must leave no account behind.
func TestVerifyLoginCode_ProvisionsAccountOnFirstRedemption(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo())
	f.seedCode("newcomer@corp.com", cfg)

	resp, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "newcomer@corp.com", Code: "123456"}, "")

	require.NoError(t, err)
	assert.Equal(t, 1, f.users.created)
	assert.Equal(t, "newcomer@corp.com", resp.User.Email)
	require.Len(t, f.bus.events, 1)
	assert.IsType(t, iamEvent.UserRegistered{}, f.bus.events[0])
}

func TestVerifyLoginCode_RefusesUnknownAddressWithoutSelfSignup(t *testing.T) {
	cfg := testAuthConfig()
	cfg.SelfSignup = false
	f := newVerifyFixture(t, cfg, newFakeUserRepo())
	f.seedCode("newcomer@corp.com", cfg)

	_, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "newcomer@corp.com", Code: "123456"}, "")

	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
	assert.Equal(t, 0, f.users.created)
}

func TestVerifyLoginCode_RefusesInactiveAccount(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusSuspended,
	}))
	f.seedCode("user@corp.com", cfg)

	_, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456"}, "")

	assert.ErrorIs(t, err, domainErr.ErrForbidden)
}

// Pairing at the moment of login is the decision under test: the mailbox has
// just been proven and the machine is present.
func TestVerifyLoginCode_PairsDeviceAndNotifiesOnFirstSight(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))
	f.seedCode("user@corp.com", cfg)

	resp, err := f.uc.Execute(context.Background(), dto.VerifyLoginCodeRequest{
		Email:  "user@corp.com",
		Code:   "123456",
		Device: &dto.DeviceInfo{Fingerprint: "fingerprint-abcdef0123", Name: "Şube PC", Platform: "windows"},
	}, "")

	require.NoError(t, err)
	require.NotNil(t, resp.Device)
	assert.True(t, resp.Device.IsNew)
	assert.Equal(t, "windows", resp.Device.Platform)

	devices, err := f.devices.ListByUser(context.Background(), resp.User.ID)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "fingerprint-abcdef0123", devices[0].Fingerprint)

	assert.Eventually(t, func() bool {
		for _, msg := range f.sender.messages() {
			if msg.Tags["category"] == "new_device" {
				return true
			}
		}
		return false
	}, time.Second, 10*time.Millisecond, "the first sight of a machine must warn the account owner")
}

// A familiar machine is not news, and mailing about it every morning trains
// users to ignore the warning that matters.
func TestVerifyLoginCode_KnownDeviceIsNotAnnouncedAgain(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))

	device := dto.DeviceInfo{Fingerprint: "fingerprint-abcdef0123", Name: "Şube PC", Platform: "windows"}

	f.seedCode("user@corp.com", cfg)
	first, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456", Device: &device}, "")
	require.NoError(t, err)
	require.True(t, first.Device.IsNew)

	f.clock = f.clock.Add(time.Hour)
	f.seedCode("user@corp.com", cfg)
	second, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456", Device: &device}, "")
	require.NoError(t, err)

	assert.False(t, second.Device.IsNew)
	assert.Equal(t, first.Device.ID, second.Device.ID, "the same machine keeps one record")
}

func TestVerifyLoginCode_RefusesRevokedDevice(t *testing.T) {
	cfg := testAuthConfig()
	user := &model.User{Email: "user@corp.com", Status: model.UserStatusActive}
	f := newVerifyFixture(t, cfg, newFakeUserRepo(user))

	device := dto.DeviceInfo{Fingerprint: "fingerprint-abcdef0123", Name: "Şube PC", Platform: "windows"}

	f.seedCode("user@corp.com", cfg)
	first, err := f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456", Device: &device}, "")
	require.NoError(t, err)

	require.NoError(t, f.devices.Revoke(context.Background(), user.ID, first.Device.ID, f.clock))

	f.clock = f.clock.Add(time.Hour)
	f.seedCode("user@corp.com", cfg)
	_, err = f.uc.Execute(context.Background(),
		dto.VerifyLoginCodeRequest{Email: "user@corp.com", Code: "123456", Device: &device}, "")

	assert.ErrorIs(t, err, domainErr.ErrForbidden)
}

// The web panel has no hardware identity and must still be able to sign in.
func TestVerifyLoginCode_WorksWithoutDeviceInfo(t *testing.T) {
	cfg := testAuthConfig()
	f := newVerifyFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))
	f.seedCode("user@corp.com", cfg)

	resp, err := f.uc.Execute(context.Background(), dto.VerifyLoginCodeRequest{
		Email:  "user@corp.com",
		Code:   "123456",
		Device: &dto.DeviceInfo{},
	}, "")

	require.NoError(t, err)
	assert.Nil(t, resp.Device)
	assert.NotEmpty(t, resp.Token)
}
