package model

import "errors"

// Errors returned by senders and by Message.Validate.
var (
	ErrNoRecipient = errors.New("notification: message has no recipient")
	ErrNoSubject   = errors.New("notification: message has no subject")
	ErrNoBody      = errors.New("notification: message has no body")

	// ErrNotConfigured means no delivery adapter is wired up. Callers must
	// treat this as a hard failure: silently swallowing it would leave a user
	// waiting for a code that was never sent.
	ErrNotConfigured = errors.New("notification: no e-mail provider configured")

	// ErrDeliveryFailed means the provider rejected or could not accept the
	// message.
	ErrDeliveryFailed = errors.New("notification: delivery failed")
)
