package service

import (
	"context"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
)

// Sender delivers a transactional e-mail.
//
// This is the only e-mail contract the application layer is allowed to depend
// on. Resend lives behind it as one adapter among possible others
// (internal/infrastructure/email/resend); swapping providers must not require
// touching a use case.
type Sender interface {
	// Send delivers the message and returns the provider's message identifier,
	// which is what makes a delivery traceable in the provider's dashboard when
	// a user reports a missing code.
	Send(ctx context.Context, msg model.Message) (messageID string, err error)

	// Name identifies the active adapter for logs and health output.
	Name() string
}
