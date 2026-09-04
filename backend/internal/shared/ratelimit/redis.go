package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter is a fixed-window limiter backed by Redis, so the window is
// shared across every instance of the API.
type RedisLimiter struct {
	client *redis.Client
	prefix string
}

// NewRedisLimiter creates a limiter over the given client.
func NewRedisLimiter(client *redis.Client, prefix string) *RedisLimiter {
	return &RedisLimiter{client: client, prefix: prefix}
}

// Allow implements Limiter.
func (l *RedisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 {
		return true, 0, nil
	}

	redisKey := fmt.Sprintf("%s:%s", l.prefix, key)

	pipe := l.client.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	// NX so that only the first hit of a window sets the deadline; refreshing
	// it on every hit would turn a fixed window into a sliding ban that never
	// expires under sustained traffic.
	pipe.ExpireNX(ctx, redisKey, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, fmt.Errorf("ratelimit: redis: %w", err)
	}

	count := incr.Val()
	if count <= int64(limit) {
		return true, 0, nil
	}

	ttl, err := l.client.TTL(ctx, redisKey).Result()
	if err != nil || ttl < 0 {
		ttl = window
	}
	return false, ttl, nil
}
