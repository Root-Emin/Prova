package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryLimiter_AllowsUpToLimit(t *testing.T) {
	l := NewMemoryLimiter()

	for i := 1; i <= 3; i++ {
		allowed, _, err := l.Allow(context.Background(), "k", 3, time.Minute)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i)
	}

	allowed, retryAfter, err := l.Allow(context.Background(), "k", 3, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Positive(t, retryAfter)
}

func TestMemoryLimiter_KeysAreIndependent(t *testing.T) {
	l := NewMemoryLimiter()

	_, _, err := l.Allow(context.Background(), "a", 1, time.Minute)
	require.NoError(t, err)
	allowed, _, err := l.Allow(context.Background(), "a", 1, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, _, err = l.Allow(context.Background(), "b", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestMemoryLimiter_WindowResets(t *testing.T) {
	l := NewMemoryLimiter()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }

	_, _, err := l.Allow(context.Background(), "k", 1, time.Minute)
	require.NoError(t, err)
	allowed, _, err := l.Allow(context.Background(), "k", 1, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)

	now = now.Add(2 * time.Minute)
	allowed, _, err = l.Allow(context.Background(), "k", 1, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
}

// A refused request still counts. Otherwise a client hammering a blocked key
// would drift back under the limit while it is being blocked.
func TestMemoryLimiter_RefusedRequestsStillCount(t *testing.T) {
	l := NewMemoryLimiter()

	for i := 0; i < 5; i++ {
		_, _, err := l.Allow(context.Background(), "k", 1, time.Minute)
		require.NoError(t, err)
	}

	l.mu.Lock()
	count := l.windows["k"].count
	l.mu.Unlock()
	assert.Equal(t, 5, count)
}

// A sweep across attacker-chosen addresses must not grow the map forever.
func TestMemoryLimiter_EvictsFinishedWindows(t *testing.T) {
	l := NewMemoryLimiter()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }

	for i := 0; i < 100; i++ {
		_, _, err := l.Allow(context.Background(), string(rune('a'+i%26))+time.Duration(i).String(), 5, time.Minute)
		require.NoError(t, err)
	}
	require.NotEmpty(t, l.windows)

	now = now.Add(2 * time.Minute)
	_, _, err := l.Allow(context.Background(), "trigger", 5, time.Minute)
	require.NoError(t, err)

	l.mu.Lock()
	remaining := len(l.windows)
	l.mu.Unlock()
	assert.Equal(t, 1, remaining, "only the window just created should remain")
}
