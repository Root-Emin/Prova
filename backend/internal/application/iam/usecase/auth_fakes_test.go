package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
)

// --- users ---

type fakeUserRepo struct {
	mu      sync.Mutex
	byID    map[uuid.UUID]*model.User
	created int
}

func newFakeUserRepo(users ...*model.User) *fakeUserRepo {
	r := &fakeUserRepo{byID: map[uuid.UUID]*model.User{}}
	for _, u := range users {
		if u.ID == uuid.Nil {
			u.ID = uuid.New()
		}
		r.byID[u.ID] = u
	}
	return r
}

func (r *fakeUserRepo) Create(_ context.Context, user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now().UTC()
	r.byID[user.ID] = user
	r.created++
	return nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.byID {
		if model.NormalizeEmail(u.Email) == model.NormalizeEmail(email) {
			return u, nil
		}
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
}

func (r *fakeUserRepo) Update(_ context.Context, user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[user.ID] = user
	return nil
}

func (r *fakeUserRepo) MarkEmailVerified(_ context.Context, id uuid.UUID, normalizedEmail string, at time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.byID[id]
	if !ok || model.NormalizeEmail(user.Email) != model.NormalizeEmail(normalizedEmail) || user.EmailVerifiedAt != nil {
		return false, nil
	}
	verifiedAt := at
	user.EmailVerifiedAt = &verifiedAt
	if user.Status == model.UserStatusInactive {
		user.Status = model.UserStatusActive
	}
	return true, nil
}

func (r *fakeUserRepo) ListDuePurge(_ context.Context, now time.Time, limit int) ([]*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var due []*model.User
	for _, u := range r.byID {
		if u.DeletionScheduledAt != nil && !u.DeletionScheduledAt.After(now) && u.DeletedAt == nil {
			due = append(due, u)
		}
	}
	return due, nil
}

func (r *fakeUserRepo) Purge(_ context.Context, id uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id]
	if !ok || u.DeletedAt != nil {
		return domainErr.New(domainErr.ErrNotFound, "user not found", nil)
	}
	u.Email, u.FirstName, u.LastName = "", "", ""
	u.EmailVerifiedAt = nil
	u.Status = model.UserStatusDeleted
	deletedAt := at
	u.DeletedAt = &deletedAt
	return nil
}

func (r *fakeUserRepo) RecordFailedAttempt(_ context.Context, id uuid.UUID, threshold int, lockUntil time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id]
	if !ok {
		return 0, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
	}
	u.FailedAttempts++
	if u.FailedAttempts >= threshold {
		locked := lockUntil
		u.LockedUntil = &locked
	}
	return u.FailedAttempts, nil
}

func (r *fakeUserRepo) ClearFailedAttempts(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		u.FailedAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byID, id)
	return nil
}

func (r *fakeUserRepo) List(_ context.Context, _, _ int) ([]*model.User, int, error) {
	return nil, 0, nil
}

// --- login codes ---

type fakeCodeRepo struct {
	mu    sync.Mutex
	codes []*model.LoginCode
}

func newFakeCodeRepo() *fakeCodeRepo { return &fakeCodeRepo{} }

func (r *fakeCodeRepo) Create(_ context.Context, code *model.LoginCode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	stored := *code
	r.codes = append(r.codes, &stored)
	return nil
}

func (r *fakeCodeRepo) GetLatestActive(_ context.Context, email string, purpose model.LoginCodePurpose, now time.Time) (*model.LoginCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.codes) - 1; i >= 0; i-- {
		c := r.codes[i]
		if c.Email == email && c.Purpose == purpose && c.IsRedeemable(now) {
			return c, nil
		}
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "login code not found", nil)
}

func (r *fakeCodeRepo) GetLatest(_ context.Context, email string, purpose model.LoginCodePurpose) (*model.LoginCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.codes) - 1; i >= 0; i-- {
		if c := r.codes[i]; c.Email == email && c.Purpose == purpose {
			return c, nil
		}
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "login code not found", nil)
}

func (r *fakeCodeRepo) IncrementAttempts(_ context.Context, id uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.codes {
		if c.ID == id {
			c.Attempts++
			return c.Attempts, nil
		}
	}
	return 0, domainErr.New(domainErr.ErrNotFound, "login code not found", nil)
}

func (r *fakeCodeRepo) MarkConsumed(_ context.Context, id uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.codes {
		if c.ID == id && c.ConsumedAt == nil {
			consumed := at
			c.ConsumedAt = &consumed
		}
	}
	return nil
}

func (r *fakeCodeRepo) InvalidateActive(_ context.Context, email string, purpose model.LoginCodePurpose, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.codes {
		if c.Email == email && c.Purpose == purpose && c.ConsumedAt == nil && c.ExpiresAt.After(at) {
			consumed := at
			c.ConsumedAt = &consumed
		}
	}
	return nil
}

func (r *fakeCodeRepo) CountSince(_ context.Context, email string, purpose model.LoginCodePurpose, since time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int
	for _, c := range r.codes {
		if c.Email == email && c.Purpose == purpose && !c.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

func (r *fakeCodeRepo) DeleteExpiredBefore(context.Context, time.Time) (int64, error) { return 0, nil }

func (r *fakeCodeRepo) DeleteByEmail(_ context.Context, email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.codes[:0]
	for _, c := range r.codes {
		if !strings.EqualFold(c.Email, email) {
			kept = append(kept, c)
		}
	}
	r.codes = kept
	return nil
}

// --- devices ---

type fakeDeviceRepo struct {
	mu      sync.Mutex
	devices []*model.Device
}

func newFakeDeviceRepo() *fakeDeviceRepo { return &fakeDeviceRepo{} }

func (r *fakeDeviceRepo) Pair(_ context.Context, device *model.Device) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.devices {
		if d.UserID == device.UserID && d.Fingerprint == device.Fingerprint {
			d.LastSeenAt = device.LastSeenAt
			*device = *d
			return false, nil
		}
	}
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	device.CreatedAt = device.LastSeenAt
	stored := *device
	r.devices = append(r.devices, &stored)
	return true, nil
}

func (r *fakeDeviceRepo) GetByFingerprint(_ context.Context, userID uuid.UUID, fingerprint string) (*model.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.devices {
		if d.UserID == userID && d.Fingerprint == fingerprint {
			return d, nil
		}
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "device not found", nil)
}

func (r *fakeDeviceRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*model.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*model.Device
	for _, d := range r.devices {
		if d.UserID == userID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (r *fakeDeviceRepo) Revoke(_ context.Context, userID, deviceID uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.devices {
		if d.ID == deviceID && d.UserID == userID {
			revoked := at
			d.RevokedAt = &revoked
			return nil
		}
	}
	return domainErr.New(domainErr.ErrNotFound, "device not found", nil)
}

func (r *fakeDeviceRepo) TouchLastSeen(context.Context, uuid.UUID, time.Time) error { return nil }

func (r *fakeDeviceRepo) GetByID(_ context.Context, userID, deviceID uuid.UUID) (*model.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.devices {
		if d.ID == deviceID && d.UserID == userID {
			return d, nil
		}
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "device not found", nil)
}

func (r *fakeDeviceRepo) DeleteByUser(_ context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.devices[:0]
	for _, d := range r.devices {
		if d.UserID != userID {
			kept = append(kept, d)
		}
	}
	r.devices = kept
	return nil
}

// --- login code service ---

// fixedCodeService always mints the same code, so tests can assert on it.
type fixedCodeService struct{ code string }

func (s fixedCodeService) Generate() (string, string, error) {
	return s.code, digestOf(s.code), nil
}

func (s fixedCodeService) Matches(digest, code string) bool {
	return digest != "" && code != "" && digest == digestOf(code)
}

func digestOf(code string) string {
	sum := sha256.Sum256([]byte("test-pepper:" + code))
	return hex.EncodeToString(sum[:])
}

// --- sender ---

type fakeSender struct {
	mu       sync.Mutex
	sent     []notifyModel.Message
	failWith error
}

func newFakeSender() *fakeSender { return &fakeSender{} }

func (s *fakeSender) Send(_ context.Context, msg notifyModel.Message) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return "", s.failWith
	}
	s.sent = append(s.sent, msg)
	return "msg_" + uuid.NewString(), nil
}

func (s *fakeSender) Name() string { return "fake" }

func (s *fakeSender) messages() []notifyModel.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]notifyModel.Message, len(s.sent))
	copy(out, s.sent)
	return out
}

// --- rate limiter ---

type fakeLimiter struct {
	mu      sync.Mutex
	counts  map[string]int
	denyAll bool
	failErr error
}

func newFakeLimiter() *fakeLimiter { return &fakeLimiter{counts: map[string]int{}} }

func (l *fakeLimiter) Allow(_ context.Context, key string, limit int, _ time.Duration) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failErr != nil {
		return false, 0, l.failErr
	}
	l.counts[key]++
	if l.denyAll || l.counts[key] > limit {
		return false, 30 * time.Second, nil
	}
	return true, 0, nil
}

// --- auth service ---

type fakeAuthService struct {
	failWith error
	// lastClaims, token'a hangi claim'lerin yazıldığını testin görebilmesi
	// için saklanır. org_id'nin doğru dolduğu ancak buradan doğrulanabilir.
	lastClaims *service.TokenClaims
}

func (s *fakeAuthService) GenerateToken(_ context.Context, claims service.TokenClaims) (string, error) {
	if s.failWith != nil {
		return "", s.failWith
	}
	s.lastClaims = &claims
	return "token-for-" + claims.UserID.String(), nil
}

func (s *fakeAuthService) ValidateToken(context.Context, string) (*service.TokenClaims, error) {
	return nil, errors.New("not implemented")
}

// fakeMembershipResolver, sabit bir organizasyon döndürür.
type fakeMembershipResolver struct {
	orgID uuid.UUID
	err   error
	calls int
}

func (r *fakeMembershipResolver) ResolveActiveOrg(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	r.calls++
	return r.orgID, r.err
}

// --- event bus ---

type fakeEventBus struct {
	mu     sync.Mutex
	events []events.Event
}

func newFakeEventBus() *fakeEventBus { return &fakeEventBus{} }

func (b *fakeEventBus) Publish(_ context.Context, _ string, event events.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}

func (b *fakeEventBus) Subscribe(string, events.Handler) {}
func (b *fakeEventBus) Close() error                     { return nil }
