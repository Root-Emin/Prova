package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		CodeLength:          6,
		CodePepper:          "test-pepper-long-enough",
		CodeTTL:             10 * time.Minute,
		MaxAttempts:         5,
		ResendCooldown:      60 * time.Second,
		MaxRequestsPerEmail: 5,
		MaxRequestsPerIP:    20,
		RateLimitWindow:     time.Hour,
		MinResponseTime:     400 * time.Millisecond,
	}
}

type requestFixture struct {
	uc      *RequestLoginCodeUseCase
	users   *fakeUserRepo
	codes   *fakeCodeRepo
	sender  *fakeSender
	limiter *fakeLimiter
	slept   time.Duration
	clock   time.Time
}

func newRequestFixture(t *testing.T, cfg config.AuthConfig, users *fakeUserRepo) *requestFixture {
	t.Helper()

	f := &requestFixture{
		users:   users,
		codes:   newFakeCodeRepo(),
		sender:  newFakeSender(),
		limiter: newFakeLimiter(),
		clock:   time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
	}
	for _, user := range users.byID {
		if user.EmailVerifiedAt == nil {
			verifiedAt := f.clock.Add(-time.Hour)
			user.EmailVerifiedAt = &verifiedAt
		}
	}
	f.uc = NewRequestLoginCodeUseCase(RequestDeps{
		Users:   f.users,
		Codes:   f.codes,
		CodeSvc: fixedCodeService{code: "123456"},
		Sender:  f.sender,
		Limiter: f.limiter,
		Cfg:     cfg,
		Log:     discardLogger(),
	})
	// A real clock would make the padding assertions flaky and slow. These
	// hooks let the test observe how long the use case *intended* to wait.
	f.uc.now = func() time.Time { return f.clock }
	f.uc.sleep = func(d time.Duration) { f.slept += d }
	return f
}

// The whole point of this endpoint's design: a caller cannot tell a registered
// address from an unknown one.
func TestRequestLoginCode_ResponseIsIdenticalForKnownAndUnknownAddress(t *testing.T) {
	cfg := testAuthConfig()

	known := newRequestFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "known@corp.com", Status: model.UserStatusActive,
	}))
	unknown := newRequestFixture(t, cfg, newFakeUserRepo())

	knownResp, err := known.uc.Execute(context.Background(),
		dto.RequestLoginCodeRequest{Email: "known@corp.com"}, "203.0.113.1")
	require.NoError(t, err)

	unknownResp, err := unknown.uc.Execute(context.Background(),
		dto.RequestLoginCodeRequest{Email: "nobody@corp.com"}, "203.0.113.1")
	require.NoError(t, err)

	assert.Equal(t, knownResp, unknownResp, "the response body must not reveal account existence")
}

// Registration is separate, so an unknown address gets no login mail but must
// still get the same answer after the same delay.
func TestRequestLoginCode_UnknownAddressIsSilent(t *testing.T) {
	cfg := testAuthConfig()
	f := newRequestFixture(t, cfg, newFakeUserRepo())

	resp, err := f.uc.Execute(context.Background(),
		dto.RequestLoginCodeRequest{Email: "nobody@corp.com"}, "203.0.113.1")

	require.NoError(t, err)
	assert.True(t, resp.Sent)
	assert.Empty(t, f.sender.messages(), "no mail may be sent to an address with no account")
	assert.Equal(t, cfg.MinResponseTime, f.slept, "the silent path must be padded to the same floor")
}

// Timing is the other half of enumeration. Both paths are stretched to the same
// floor, so the fast path cannot be identified by a stopwatch.
func TestRequestLoginCode_PadsResponseToFloor(t *testing.T) {
	cfg := testAuthConfig()
	f := newRequestFixture(t, cfg, newFakeUserRepo())

	// Simulate the work taking 150ms of real time.
	start := f.clock
	f.uc.now = func() time.Time { return f.clock }
	f.uc.sleep = func(d time.Duration) { f.slept += d }
	f.clock = start

	_, err := f.uc.Execute(context.Background(),
		dto.RequestLoginCodeRequest{Email: "someone@corp.com"}, "203.0.113.1")
	require.NoError(t, err)

	assert.Equal(t, cfg.MinResponseTime, f.slept)
}

func TestRequestLoginCode_SendsCodeAndStoresOnlyTheDigest(t *testing.T) {
	f := newRequestFixture(t, testAuthConfig(), newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))

	_, err := f.uc.Execute(context.Background(),
		dto.RequestLoginCodeRequest{Email: "User@Corp.com "}, "203.0.113.1")
	require.NoError(t, err)

	messages := f.sender.messages()
	require.Len(t, messages, 1)
	assert.Equal(t, "user@corp.com", messages[0].To.Email, "the address must be normalized before use")
	assert.Contains(t, messages[0].TextBody, "123456")
	assert.NotContains(t, messages[0].Subject, "123456", "the code must stay out of the subject line")

	require.Len(t, f.codes.codes, 1)
	stored := f.codes.codes[0]
	assert.Equal(t, "user@corp.com", stored.Email)
	assert.Equal(t, digestOf("123456"), stored.CodeDigest)
	assert.NotContains(t, stored.CodeDigest, "123456", "the plaintext code must never be stored")
	assert.Equal(t, 5, stored.MaxAttempts)
	assert.Equal(t, f.clock.Add(10*time.Minute), stored.ExpiresAt)
}

// A second request retires the first code. Several live codes for one address
// would multiply the guessing surface for the whole TTL.
func TestRequestLoginCode_RetiresPreviousCode(t *testing.T) {
	cfg := testAuthConfig()
	f := newRequestFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))

	_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "")
	require.NoError(t, err)

	// Move past the cooldown and ask again.
	f.clock = f.clock.Add(2 * time.Minute)
	_, err = f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "")
	require.NoError(t, err)

	require.Len(t, f.codes.codes, 2)
	assert.NotNil(t, f.codes.codes[0].ConsumedAt, "the earlier code must be burned")
	assert.Nil(t, f.codes.codes[1].ConsumedAt)
}

func TestRequestLoginCode_EnforcesResendCooldown(t *testing.T) {
	f := newRequestFixture(t, testAuthConfig(), newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))

	_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "")
	require.NoError(t, err)

	f.clock = f.clock.Add(10 * time.Second)
	_, err = f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "")

	require.Error(t, err)
	assert.ErrorIs(t, err, domainErr.ErrRateLimited)
	assert.Len(t, f.sender.messages(), 1, "the cooldown must stop the second mail")
}

func TestRequestLoginCode_EnforcesPerAddressLimit(t *testing.T) {
	cfg := testAuthConfig()
	cfg.ResendCooldown = 0
	cfg.MaxRequestsPerEmail = 2

	f := newRequestFixture(t, cfg, newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))

	for i := 0; i < 2; i++ {
		_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "")
		require.NoError(t, err)
	}

	_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "")
	assert.ErrorIs(t, err, domainErr.ErrRateLimited)
}

// The IP limit is the only one that constrains a sweep across many addresses,
// so it must be charged before any address-specific work.
func TestRequestLoginCode_EnforcesPerIPLimit(t *testing.T) {
	cfg := testAuthConfig()
	cfg.ResendCooldown = 0
	cfg.MaxRequestsPerIP = 2

	f := newRequestFixture(t, cfg, newFakeUserRepo())

	for i, email := range []string{"a@corp.com", "b@corp.com"} {
		_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: email}, "203.0.113.9")
		require.NoError(t, err, "request %d", i)
	}

	_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "c@corp.com"}, "203.0.113.9")
	assert.ErrorIs(t, err, domainErr.ErrRateLimited)
}

// A limiter outage must not take down the only way into the product.
func TestRequestLoginCode_FailsOpenWhenLimiterIsBroken(t *testing.T) {
	f := newRequestFixture(t, testAuthConfig(), newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))
	f.limiter.failErr = errors.New("redis down")

	_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "203.0.113.1")

	require.NoError(t, err)
	assert.Len(t, f.sender.messages(), 1)
}

// A code nobody can read is the same as no code. The caller must be told.
func TestRequestLoginCode_FailsWhenDeliveryFails(t *testing.T) {
	f := newRequestFixture(t, testAuthConfig(), newFakeUserRepo(&model.User{
		Email: "user@corp.com", Status: model.UserStatusActive,
	}))
	f.sender.failWith = notifyModel.ErrNotConfigured

	_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "user@corp.com"}, "")

	require.Error(t, err)
	assert.ErrorIs(t, err, domainErr.ErrInternal)
}

func TestRequestLoginCode_RejectsEmptyAddress(t *testing.T) {
	f := newRequestFixture(t, testAuthConfig(), newFakeUserRepo())

	_, err := f.uc.Execute(context.Background(), dto.RequestLoginCodeRequest{Email: "   "}, "")

	assert.ErrorIs(t, err, domainErr.ErrValidation)
}

// Invited (inactive/unverified) accounts must receive a login code so first
// launch can complete. Unknown addresses remain silent.
func TestRequestLoginCode_SendsToUnverifiedInvitedUser(t *testing.T) {
	cfg := testAuthConfig()
	user := &model.User{Email: "invited@corp.com", Status: model.UserStatusInactive}
	f := newRequestFixture(t, cfg, newFakeUserRepo(user))
	user.EmailVerifiedAt = nil

	resp, err := f.uc.Execute(context.Background(),
		dto.RequestLoginCodeRequest{Email: "invited@corp.com"}, "203.0.113.1")

	require.NoError(t, err)
	assert.True(t, resp.Sent)
	require.Len(t, f.sender.messages(), 1)
	assert.Equal(t, "invited@corp.com", f.sender.messages()[0].To.Email)
}

func TestRequestLoginCode_SuspendedUserIsSilent(t *testing.T) {
	cfg := testAuthConfig()
	user := &model.User{Email: "locked@corp.com", Status: model.UserStatusSuspended}
	f := newRequestFixture(t, cfg, newFakeUserRepo(user))

	resp, err := f.uc.Execute(context.Background(),
		dto.RequestLoginCodeRequest{Email: "locked@corp.com"}, "203.0.113.1")

	require.NoError(t, err)
	assert.True(t, resp.Sent)
	assert.Empty(t, f.sender.messages())
}
