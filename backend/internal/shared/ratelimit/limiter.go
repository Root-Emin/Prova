// Package ratelimit provides a fixed-window request limiter.
//
// It exists as a shared port because the same counter is needed in several
// places with different backing stores: login-code requests today, GraphQL
// operations later.
package ratelimit

import (
	"context"
	"time"
)

// Limiter counts events per key inside a fixed window.
type Limiter interface {
	// Allow records one event against the key and reports whether it stays
	// within limit. When it does not, retryAfter says how long the caller
	// should wait.
	//
	// The event is counted even when it is refused: a client that keeps
	// hammering a refused key must not reset itself back under the limit by
	// waiting for individual counts to drop off.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error)
}
