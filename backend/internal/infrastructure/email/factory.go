// Package email selects and builds the transactional e-mail adapter.
//
// This package is the only place in the codebase that names a provider. Use
// cases depend on notification/service.Sender; main.go asks this factory for an
// implementation. Adding a provider means adding a sub-package and one case
// below.
package email

import (
	"context"
	"fmt"
	"strings"

	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	notify "github.com/masterfabric-go/masterfabric/internal/domain/notification/service"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/email/resend"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

// Provider names accepted in EMAIL_PROVIDER.
const (
	ProviderResend = "resend"
	ProviderNone   = "none"
)

// New builds the sender named by cfg.Provider.
func New(cfg config.EmailConfig) (notify.Sender, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case ProviderResend, "":
		return resend.New(cfg)
	case ProviderNone:
		return Disabled{}, nil
	default:
		return nil, fmt.Errorf("email: unknown provider %q", cfg.Provider)
	}
}

// Disabled is the sender used when EMAIL_PROVIDER=none.
//
// It refuses to send rather than printing the message to the console. Codes
// reach real mailboxes or the request fails: a console-printed code is a
// credential in the log stream, and a developer who can read it is a developer
// who never notices that delivery is broken.
type Disabled struct{}

// Name implements service.Sender.
func (Disabled) Name() string { return ProviderNone }

// Send implements service.Sender and always fails.
func (Disabled) Send(context.Context, notifyModel.Message) (string, error) {
	return "", notifyModel.ErrNotConfigured
}
