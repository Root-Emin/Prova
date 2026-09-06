package usecase

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

type fakeEmailVerificationRepo struct {
	mu      sync.Mutex
	current map[uuid.UUID]*model.EmailVerificationChallenge
	failErr error
}

func newFakeEmailVerificationRepo() *fakeEmailVerificationRepo {
	return &fakeEmailVerificationRepo{current: map[uuid.UUID]*model.EmailVerificationChallenge{}}
}

func (r *fakeEmailVerificationRepo) Save(_ context.Context, challenge *model.EmailVerificationChallenge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failErr != nil {
		return r.failErr
	}
	copy := *challenge
	r.current[challenge.UserID] = &copy
	return nil
}

func (r *fakeEmailVerificationRepo) Get(_ context.Context, userID uuid.UUID) (*model.EmailVerificationChallenge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failErr != nil {
		return nil, r.failErr
	}
	challenge, ok := r.current[userID]
	if !ok {
		return nil, domainErr.New(domainErr.ErrNotFound, "challenge not found", nil)
	}
	copy := *challenge
	return &copy, nil
}

func (r *fakeEmailVerificationRepo) GetActive(ctx context.Context, userID uuid.UUID, now time.Time) (*model.EmailVerificationChallenge, error) {
	challenge, err := r.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !challenge.IsActive(now) {
		return nil, domainErr.New(domainErr.ErrNotFound, "challenge not found", nil)
	}
	return challenge, nil
}

func (r *fakeEmailVerificationRepo) IncrementAttempts(_ context.Context, userID, challengeID uuid.UUID, now time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failErr != nil {
		return 0, r.failErr
	}
	challenge, ok := r.current[userID]
	if !ok || challenge.ID != challengeID || !challenge.IsActive(now) {
		return 0, domainErr.New(domainErr.ErrNotFound, "challenge not found", nil)
	}
	challenge.Attempts++
	return challenge.Attempts, nil
}

func (r *fakeEmailVerificationRepo) Consume(_ context.Context, userID, challengeID uuid.UUID, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failErr != nil {
		return false, r.failErr
	}
	challenge, ok := r.current[userID]
	if !ok || challenge.ID != challengeID || !challenge.IsActive(now) {
		return false, nil
	}
	delete(r.current, userID)
	return true, nil
}

func (r *fakeEmailVerificationRepo) Delete(_ context.Context, userID, challengeID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failErr != nil {
		return r.failErr
	}
	if challenge := r.current[userID]; challenge != nil && challenge.ID == challengeID {
		delete(r.current, userID)
	}
	return nil
}

type fixedBoundCodeService struct {
	mu    sync.Mutex
	codes []string
	next  int
}

func newFixedBoundCodeService(codes ...string) *fixedBoundCodeService {
	return &fixedBoundCodeService{codes: codes}
}

func (s *fixedBoundCodeService) Generate() (string, string, error) {
	return "", "", errors.New("unbound generation is forbidden in verification tests")
}

func (s *fixedBoundCodeService) Matches(string, string) bool { return false }

func (s *fixedBoundCodeService) GenerateFor(userID uuid.UUID, email string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.next >= len(s.codes) {
		return "", "", errors.New("no fixed code left")
	}
	code := s.codes[s.next]
	s.next++
	return code, verificationDigest(userID, email, code), nil
}

func (s *fixedBoundCodeService) MatchesFor(userID uuid.UUID, email, digest, code string) bool {
	want, err := hex.DecodeString(verificationDigest(userID, email, code))
	if err != nil {
		return false
	}
	got, err := hex.DecodeString(digest)
	return err == nil && hmac.Equal(want, got)
}

func verificationDigest(userID uuid.UUID, email, code string) string {
	mac := hmac.New(sha256.New, []byte("verification-test-pepper"))
	_, _ = mac.Write([]byte(userID.String() + ":" + model.NormalizeEmail(email) + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func testEmailVerificationConfig() config.EmailVerificationConfig {
	return config.EmailVerificationConfig{
		Pepper: "verification-test-pepper", OTPTTL: 5 * time.Minute,
		MaxAttempts: 5, ResendCooldown: time.Minute,
		MaxRequestsPerEmail: 5, MaxRequestsPerIP: 20, RateLimitWindow: time.Hour,
	}
}

type emailVerificationFixture struct {
	request *RequestEmailVerificationCodeUseCase
	verify  *VerifyEmailUseCase
	users   *fakeUserRepo
	repo    *fakeEmailVerificationRepo
	sender  *fakeSender
	limiter *fakeLimiter
	codes   *fixedBoundCodeService
	clock   time.Time
}

func newEmailVerificationFixture(user *model.User, codes ...string) *emailVerificationFixture {
	users := newFakeUserRepo(user)
	f := &emailVerificationFixture{
		users: users, repo: newFakeEmailVerificationRepo(), sender: newFakeSender(),
		limiter: newFakeLimiter(), codes: newFixedBoundCodeService(codes...),
		clock: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC),
	}
	cfg := testEmailVerificationConfig()
	f.request = NewRequestEmailVerificationCodeUseCase(RequestEmailVerificationDeps{
		Users: users, Challenges: f.repo, Codes: f.codes, Sender: f.sender,
		Limiter: f.limiter, Cfg: cfg, Log: discardLogger(),
	})
	f.verify = NewVerifyEmailUseCase(VerifyEmailDeps{
		Users: users, Challenges: f.repo, Codes: f.codes, Cfg: cfg, Log: discardLogger(),
	})
	f.request.now = func() time.Time { return f.clock }
	f.verify.now = func() time.Time { return f.clock }
	return f
}

func unverifiedUser(email string) *model.User {
	return &model.User{Email: email, FirstName: "Ada", LastName: "Lovelace", Status: model.UserStatusInactive}
}

func (f *emailVerificationFixture) issue(t *testing.T, email string) *model.EmailVerificationChallenge {
	t.Helper()
	_, err := f.request.Execute(context.Background(), dto.RequestEmailVerificationCodeRequest{Email: email}, "203.0.113.8")
	require.NoError(t, err)
	user, err := f.users.GetByEmail(context.Background(), email)
	require.NoError(t, err)
	challenge, err := f.repo.Get(context.Background(), user.ID)
	require.NoError(t, err)
	return challenge
}

func TestRegister_ProvisionsUnverifiedUserWithoutSendingMail(t *testing.T) {
	users := newFakeUserRepo()
	repo := newFakeEmailVerificationRepo()
	sender := newFakeSender()
	registration := NewRegisterUseCase(users, newFakeEventBus())

	result, err := registration.Execute(context.Background(), dto.RegisterRequest{
		Email: " Ada@Example.com ", FirstName: "Ada", LastName: "Lovelace",
	}, "203.0.113.8")

	require.NoError(t, err)
	assert.True(t, result.Registered)
	user, err := users.GetByEmail(context.Background(), "ada@example.com")
	require.NoError(t, err)
	assert.Nil(t, user.EmailVerifiedAt)
	assert.Equal(t, model.UserStatusInactive, user.Status)
	assert.Empty(t, sender.messages(), "the code is sent only after the Desktop login request")
	challenge, err := repo.Get(context.Background(), user.ID)
	assert.ErrorIs(t, err, domainErr.ErrNotFound)
	assert.Nil(t, challenge)
}

func TestRegister_DoesNotDependOnEmailDelivery(t *testing.T) {
	users := newFakeUserRepo()
	registration := NewRegisterUseCase(users, nil)

	result, err := registration.Execute(context.Background(), dto.RegisterRequest{
		Email: "ada@example.com", FirstName: "Ada", LastName: "Lovelace",
	}, "")

	require.NoError(t, err)
	assert.True(t, result.Registered)
	user, getErr := users.GetByEmail(context.Background(), "ada@example.com")
	require.NoError(t, getErr)
	assert.Nil(t, user.EmailVerifiedAt)
}

func TestRegister_IsIdempotentForExistingUnverifiedUser(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	users := newFakeUserRepo(user)
	registration := NewRegisterUseCase(users, nil)

	result, err := registration.Execute(context.Background(), dto.RegisterRequest{
		Email: "ada@example.com", FirstName: "Ada", LastName: "Lovelace",
	}, "203.0.113.8")

	require.NoError(t, err)
	assert.True(t, result.Registered)
	assert.Equal(t, 0, users.created, "must not create a duplicate user")
	assert.Len(t, users.byID, 1)
}

func TestRegister_RejectsVerifiedExistingEmail(t *testing.T) {
	now := time.Now().UTC()
	user := unverifiedUser("ada@example.com")
	user.EmailVerifiedAt = &now
	user.Status = model.UserStatusActive
	users := newFakeUserRepo(user)
	registration := NewRegisterUseCase(users, nil)

	_, err := registration.Execute(context.Background(), dto.RegisterRequest{
		Email: "ada@example.com", FirstName: "Ada", LastName: "Lovelace",
	}, "")

	assert.ErrorIs(t, err, domainErr.ErrAlreadyExists)
}

func TestVerifyEmail_ValidCodeMarksEmailVerifiedWithoutAuthentication(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	f := newEmailVerificationFixture(user, "123456")
	f.issue(t, user.Email)

	result, err := f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: "123456"})

	require.NoError(t, err)
	assert.True(t, result.Verified)
	require.NotNil(t, user.EmailVerifiedAt)
	assert.Equal(t, f.clock, *user.EmailVerifiedAt)
	assert.Equal(t, model.UserStatusActive, user.Status)
	// VerifyEmailResponse has no token/session field by design.
	assert.NotContains(t, fmt.Sprintf("%+v", result), "token")
}

func TestVerifyEmail_WrongMalformedExpiredAndReplayCodesFail(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    string
		advance time.Duration
	}{
		{name: "wrong", code: "654321"},
		{name: "malformed", code: "12345x"},
		{name: "expired", code: "123456", advance: 5 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := unverifiedUser("ada@example.com")
			f := newEmailVerificationFixture(user, "123456")
			f.issue(t, user.Email)
			f.clock = f.clock.Add(tc.advance)
			_, err := f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: tc.code})
			assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
			assert.Nil(t, user.EmailVerifiedAt)
		})
	}

	user := unverifiedUser("replay@example.com")
	f := newEmailVerificationFixture(user, "123456")
	f.issue(t, user.Email)
	_, err := f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: "123456"})
	require.NoError(t, err)
	_, err = f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: "123456"})
	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
}

func TestVerifyEmail_FiveWrongAttemptsExhaustChallenge(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	f := newEmailVerificationFixture(user, "123456")
	challenge := f.issue(t, user.Email)

	for attempt := 1; attempt <= 5; attempt++ {
		_, err := f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: "000000"})
		assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
	}
	stored, err := f.repo.Get(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, stored.Attempts)
	assert.Equal(t, challenge.ID, stored.ID)
	_, err = f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: "123456"})
	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
	assert.Nil(t, user.EmailVerifiedAt)
}

func TestEmailVerification_NewCodeInvalidatesPreviousAndCooldownIsSixtySeconds(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	f := newEmailVerificationFixture(user, "111111", "222222")
	first := f.issue(t, user.Email)
	_, err := f.request.Execute(context.Background(), dto.RequestEmailVerificationCodeRequest{Email: user.Email}, "")
	assert.ErrorIs(t, err, domainErr.ErrRateLimited)

	f.clock = f.clock.Add(61 * time.Second)
	second := f.issue(t, user.Email)
	assert.NotEqual(t, first.ID, second.ID)
	_, err = f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: "111111"})
	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
	_, err = f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: user.Email, Code: "222222"})
	require.NoError(t, err)
}

func TestEmailVerification_IsAccountAndCurrentEmailBound(t *testing.T) {
	userA := unverifiedUser("a@example.com")
	f := newEmailVerificationFixture(userA, "123456")
	challenge := f.issue(t, userA.Email)
	userB := unverifiedUser("b@example.com")
	userB.ID = uuid.New()
	f.users.byID[userB.ID] = userB
	// Even copying A's digest/challenge under B cannot make it valid.
	copy := *challenge
	copy.UserID = userB.ID
	f.repo.current[userB.ID] = &copy
	_, err := f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: userB.Email, Code: "123456"})
	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
	assert.Nil(t, userB.EmailVerifiedAt)

	userA.Email = "new@example.com"
	_, err = f.verify.Execute(context.Background(), dto.VerifyEmailRequest{Email: "new@example.com", Code: "123456"})
	assert.ErrorIs(t, err, domainErr.ErrUnauthorized)
	assert.Nil(t, userA.EmailVerifiedAt)
}

func TestEmailVerification_RedisAndLimiterFailuresFailClosed(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	f := newEmailVerificationFixture(user, "123456")
	f.repo.failErr = errors.New("redis down")
	_, err := f.request.Execute(context.Background(), dto.RequestEmailVerificationCodeRequest{Email: user.Email}, "")
	assert.Error(t, err)
	assert.Empty(t, f.sender.messages())

	f.repo.failErr = nil
	f.limiter.failErr = errors.New("redis down")
	_, err = f.request.Execute(context.Background(), dto.RequestEmailVerificationCodeRequest{Email: user.Email}, "")
	assert.ErrorIs(t, err, domainErr.ErrInternal)
}

func TestEmailVerification_HourlySendLimitIsEnforced(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	f := newEmailVerificationFixture(user, "000001", "000002", "000003", "000004", "000005", "000006")
	f.request.Cfg.ResendCooldown = 0
	for i := 0; i < 5; i++ {
		_, err := f.request.Execute(context.Background(), dto.RequestEmailVerificationCodeRequest{Email: user.Email}, "")
		require.NoError(t, err)
	}
	_, err := f.request.Execute(context.Background(), dto.RequestEmailVerificationCodeRequest{Email: user.Email}, "")
	assert.ErrorIs(t, err, domainErr.ErrRateLimited)
	assert.Len(t, f.sender.messages(), 5)
}

func TestEmailVerification_StoredStateNeverContainsPlaintextOTP(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	f := newEmailVerificationFixture(user, "012345")
	challenge := f.issue(t, user.Email)
	assert.False(t, strings.Contains(challenge.CodeDigest, "012345"))
	assert.NotEqual(t, "012345", challenge.CodeDigest)
}

func TestEmailVerification_LogsNeverContainPlaintextOTP(t *testing.T) {
	user := unverifiedUser("ada@example.com")
	f := newEmailVerificationFixture(user, "012345")
	var logs bytes.Buffer
	f.request.Log = slog.New(slog.NewJSONHandler(&logs, nil))

	f.issue(t, user.Email)

	assert.NotContains(t, logs.String(), "012345")
	assert.NotContains(t, logs.String(), testEmailVerificationConfig().Pepper)
}
