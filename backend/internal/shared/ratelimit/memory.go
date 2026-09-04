package ratelimit

import (
	"context"
	"sync"
	"time"
)

// MemoryLimiter is a fixed-window limiter held in process memory.
//
// It is the fallback for when Redis is unavailable. The window is per instance,
// so behind several replicas the effective limit multiplies by the replica
// count — acceptable as a degraded mode, not as the production configuration.
type MemoryLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	now     func() time.Time
}

type window struct {
	count     int
	expiresAt time.Time
}

// NewMemoryLimiter creates an empty limiter.
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{windows: make(map[string]*window), now: time.Now}
}

// Allow implements Limiter.
func (l *MemoryLimiter) Allow(_ context.Context, key string, limit int, size time.Duration) (bool, time.Duration, error) {
	if limit <= 0 {
		return true, 0, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.evictExpired(now)

	w, ok := l.windows[key]
	if !ok || now.After(w.expiresAt) {
		w = &window{expiresAt: now.Add(size)}
		l.windows[key] = w
	}
	w.count++

	if w.count <= limit {
		return true, 0, nil
	}
	return false, w.expiresAt.Sub(now), nil
}

// evictExpired drops finished windows so that a stream of distinct keys — one
// per attacker-chosen address — cannot grow the map without bound.
func (l *MemoryLimiter) evictExpired(now time.Time) {
	for key, w := range l.windows {
		if now.After(w.expiresAt) {
			delete(l.windows, key)
		}
	}
}
