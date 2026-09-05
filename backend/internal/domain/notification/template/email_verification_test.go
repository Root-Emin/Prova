package template

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
)

func TestEmailVerification_CarriesCodeInBothBodies(t *testing.T) {
	msg := EmailVerification(model.Address{Email: "user@corp.com"}, "731904", 15*time.Minute)

	require.NoError(t, msg.Validate())
	assert.Equal(t, subjectEmailVerification, msg.Subject)
	assert.NotContains(t, msg.Subject, "731904")
	assert.Contains(t, msg.TextBody, "731904")
	assert.Contains(t, msg.HTMLBody, "731904")
	assert.Contains(t, msg.TextBody, "15 dakika")
	assert.Contains(t, msg.HTMLBody, "15 dakika")
}

func TestEmailVerification_UsesBrandedShellWithoutLinks(t *testing.T) {
	msg := EmailVerification(model.Address{Email: "user@corp.com"}, "731904", 15*time.Minute)

	assert.Contains(t, msg.HTMLBody, "data:image/png")
	assert.Contains(t, msg.HTMLBody, `alt="Prova"`)
	assert.Contains(t, msg.HTMLBody, brandTagline)
	assert.NotContains(t, msg.HTMLBody, "<a ")
	assert.NotContains(t, msg.HTMLBody, "http://")
	assert.NotContains(t, msg.HTMLBody, "https://")
}
