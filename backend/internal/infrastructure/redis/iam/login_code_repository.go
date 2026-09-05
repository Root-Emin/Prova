// Package iam contains Redis-backed, short-lived IAM state.
package iam

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

const (
	loginCodePrefix = "prova:iam:login-code:"
	loginEmailIndex = "prova:iam:login-email:"
	loginIDIndex    = "prova:iam:login-id:"
	loginCountKey   = "prova:iam:login-count:"
)

// LoginCodeRepo keeps passwordless-login challenges in Redis, keyed primarily by
// immutable user ID. The e-mail index is only a lookup aid; it never replaces
// the user ID in the digest or in the stored challenge identity.
type LoginCodeRepo struct {
	redis *redis.Client
}

func NewLoginCodeRepo(client *redis.Client) *LoginCodeRepo {
	return &LoginCodeRepo{redis: client}
}

type storedLoginCode struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"user_id"`
	Email       string                 `json:"email"`
	CodeDigest  string                 `json:"code_digest"`
	Purpose     model.LoginCodePurpose `json:"purpose"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	ExpiresAt   time.Time              `json:"expires_at"`
	ConsumedAt  *time.Time             `json:"consumed_at,omitempty"`
	RequestIP   string                 `json:"request_ip,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

func (r *LoginCodeRepo) Create(ctx context.Context, code *model.LoginCode) error {
	if code.UserID == uuid.Nil {
		return domainErr.New(domainErr.ErrInternal, "account-bound login code has no user id", nil)
	}
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = time.Now().UTC()
	}
	if !code.ExpiresAt.After(code.CreatedAt) {
		return domainErr.New(domainErr.ErrInternal, "login code has invalid expiry", nil)
	}

	payload, err := json.Marshal(storedLoginCodeFromModel(code))
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to encode login code", err)
	}
	ttl := time.Until(code.ExpiresAt)
	if ttl <= 0 {
		return domainErr.New(domainErr.ErrInternal, "login code expired before storage", nil)
	}

	key := userKey(code.UserID)
	pipe := r.redis.TxPipeline()
	pipe.Set(ctx, key, payload, ttl)
	pipe.Set(ctx, emailKey(code.Email), key, ttl)
	pipe.Set(ctx, idKey(code.ID), key, ttl)
	count := countKey(code.Email)
	pipe.ZAdd(ctx, count, redis.Z{Score: float64(code.CreatedAt.UnixNano()), Member: code.ID.String()})
	pipe.Expire(ctx, count, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to store login code", err)
	}
	return nil
}

func (r *LoginCodeRepo) GetLatestActive(ctx context.Context, email string, purpose model.LoginCodePurpose, now time.Time) (*model.LoginCode, error) {
	code, err := r.get(ctx, email)
	if err != nil {
		return nil, err
	}
	if code.Purpose != purpose || !code.IsRedeemable(now) {
		return nil, notFound()
	}
	return code, nil
}

func (r *LoginCodeRepo) GetLatest(ctx context.Context, email string, purpose model.LoginCodePurpose) (*model.LoginCode, error) {
	code, err := r.get(ctx, email)
	if err != nil {
		return nil, err
	}
	if code.Purpose != purpose {
		return nil, notFound()
	}
	return code, nil
}

func (r *LoginCodeRepo) IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error) {
	key, err := r.findKey(ctx, id)
	if err != nil {
		return 0, err
	}
	var attempts int
	err = r.redis.Watch(ctx, func(tx *redis.Tx) error {
		code, err := getStored(ctx, tx, key)
		if err != nil {
			return err
		}
		code.Attempts++
		attempts = code.Attempts
		return txSetStored(ctx, tx, key, code)
	}, key)
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to increment login code attempts", err)
	}
	return attempts, nil
}

func (r *LoginCodeRepo) MarkConsumed(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.MarkConsumedIfActive(ctx, id, at)
	return err
}

// MarkConsumedIfActive is the atomic one-time-use operation used by the
// verifier. A false result means another request won the race.
func (r *LoginCodeRepo) MarkConsumedIfActive(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	key, err := r.findKey(ctx, id)
	if err != nil {
		return false, err
	}
	consumed := false
	err = r.redis.Watch(ctx, func(tx *redis.Tx) error {
		code, err := getStored(ctx, tx, key)
		if err != nil {
			return err
		}
		if code.ConsumedAt != nil {
			return nil
		}
		stamp := at
		code.ConsumedAt = &stamp
		if err := txSetStored(ctx, tx, key, code); err != nil {
			return err
		}
		consumed = true
		return nil
	}, key)
	if err != nil {
		return false, domainErr.New(domainErr.ErrInternal, "failed to consume login code", err)
	}
	return consumed, nil
}

func (r *LoginCodeRepo) InvalidateActive(ctx context.Context, email string, purpose model.LoginCodePurpose, at time.Time) error {
	code, err := r.get(ctx, email)
	if errors.Is(err, domainErr.ErrNotFound) {
		return nil
	}
	if err != nil || code.Purpose != purpose || !code.IsRedeemable(at) {
		return err
	}
	_, err = r.MarkConsumedIfActive(ctx, code.ID, at)
	return err
}

func (r *LoginCodeRepo) CountSince(ctx context.Context, email string, purpose model.LoginCodePurpose, since time.Time) (int, error) {
	_ = purpose // the count key is scoped to login-code requests by design.
	n, err := r.redis.ZCount(ctx, countKey(email), strconv.FormatInt(since.UnixNano(), 10), "+inf").Result()
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to count login codes", err)
	}
	return int(n), nil
}

func (r *LoginCodeRepo) DeleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	// Individual challenges have Redis TTLs. The sorted request-count index is
	// pruned lazily by Redis expiry and never contains the OTP digest.
	return 0, nil
}

func (r *LoginCodeRepo) DeleteByEmail(ctx context.Context, email string) error {
	key, err := r.redis.Get(ctx, emailKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to find login code", err)
	}
	code, _ := r.get(ctx, email)
	keys := []string{emailKey(email), key}
	if code != nil {
		keys = append(keys, idKey(code.ID))
	}
	if err := r.redis.Del(ctx, keys...).Err(); err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete login code", err)
	}
	return nil
}

func (r *LoginCodeRepo) get(ctx context.Context, email string) (*model.LoginCode, error) {
	key, err := r.redis.Get(ctx, emailKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, notFound()
	}
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get login code index", err)
	}
	raw, err := r.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, notFound()
	}
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get login code", err)
	}
	var stored storedLoginCode
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to decode login code", err)
	}
	return stored.toModel(), nil
}

func (r *LoginCodeRepo) findKey(ctx context.Context, id uuid.UUID) (string, error) {
	key, err := r.redis.Get(ctx, idKey(id)).Result()
	if errors.Is(err, redis.Nil) {
		return "", notFound()
	}
	if err != nil {
		return "", domainErr.New(domainErr.ErrInternal, "failed to find login code", err)
	}
	return key, nil
}

func getStored(ctx context.Context, tx *redis.Tx, key string) (storedLoginCode, error) {
	raw, err := tx.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return storedLoginCode{}, notFound()
	}
	if err != nil {
		return storedLoginCode{}, err
	}
	var code storedLoginCode
	if err := json.Unmarshal(raw, &code); err != nil {
		return storedLoginCode{}, err
	}
	return code, nil
}

func txSetStored(ctx context.Context, tx *redis.Tx, key string, code storedLoginCode) error {
	payload, err := json.Marshal(code)
	if err != nil {
		return err
	}
	ttl := time.Until(code.ExpiresAt)
	if ttl <= 0 {
		return notFound()
	}
	_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, key, payload, ttl)
		return nil
	})
	return err
}

func storedLoginCodeFromModel(c *model.LoginCode) storedLoginCode {
	return storedLoginCode{ID: c.ID, UserID: c.UserID, Email: c.Email, CodeDigest: c.CodeDigest,
		Purpose: c.Purpose, Attempts: c.Attempts, MaxAttempts: c.MaxAttempts, ExpiresAt: c.ExpiresAt,
		ConsumedAt: c.ConsumedAt, RequestIP: c.RequestIP, CreatedAt: c.CreatedAt}
}

func (c storedLoginCode) toModel() *model.LoginCode {
	return &model.LoginCode{ID: c.ID, UserID: c.UserID, Email: c.Email, CodeDigest: c.CodeDigest,
		Purpose: c.Purpose, Attempts: c.Attempts, MaxAttempts: c.MaxAttempts, ExpiresAt: c.ExpiresAt,
		ConsumedAt: c.ConsumedAt, RequestIP: c.RequestIP, CreatedAt: c.CreatedAt}
}

func userKey(id uuid.UUID) string { return loginCodePrefix + id.String() }

func emailKey(email string) string { return loginEmailIndex + hashEmail(email) }

func idKey(id uuid.UUID) string { return loginIDIndex + id.String() }

func countKey(email string) string { return loginCountKey + hashEmail(email) }

func hashEmail(email string) string {
	sum := sha256.Sum256([]byte(model.NormalizeEmail(email)))
	return hex.EncodeToString(sum[:])
}

func notFound() error { return domainErr.New(domainErr.ErrNotFound, "login code not found", nil) }

var _ iamRepo.LoginCodeRepository = (*LoginCodeRepo)(nil)
var _ interface {
	MarkConsumedIfActive(context.Context, uuid.UUID, time.Time) (bool, error)
} = (*LoginCodeRepo)(nil)
