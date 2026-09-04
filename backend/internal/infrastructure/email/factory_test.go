package email

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

func TestNew_SelectsResendByDefault(t *testing.T) {
	cfg := config.EmailConfig{
		FromAddress: "noreply@mail.example.com",
		Resend:      config.ResendConfig{APIKey: "re_test_key"},
	}

	sender, err := New(cfg)

	require.NoError(t, err)
	assert.Equal(t, ProviderResend, sender.Name())
}

func TestNew_RejectsUnknownProvider(t *testing.T) {
	_, err := New(config.EmailConfig{Provider: "sendgrid"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sendgrid")
}

// EMAIL_PROVIDER=none refuses rather than printing the code to the console. A
// logged code is a credential in the log stream, and it hides the fact that
// delivery is broken.
func TestDisabled_FailsInsteadOfPrinting(t *testing.T) {
	sender, err := New(config.EmailConfig{Provider: ProviderNone})
	require.NoError(t, err)

	id, err := sender.Send(context.Background(), notifyModel.Message{
		To:       notifyModel.Address{Email: "user@corp.com"},
		Subject:  "subject",
		TextBody: "482913",
	})

	assert.Empty(t, id)
	assert.ErrorIs(t, err, notifyModel.ErrNotConfigured)
}
