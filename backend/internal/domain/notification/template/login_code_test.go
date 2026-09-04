package template

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
)

func TestLoginCode_CarriesCodeInBothBodies(t *testing.T) {
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute)

	require.NoError(t, msg.Validate())
	assert.Equal(t, "user@corp.com", msg.To.Email)
	assert.Contains(t, msg.TextBody, "482913")
	assert.Contains(t, msg.HTMLBody, "482913")
	assert.Contains(t, msg.TextBody, "10 dakika")
	assert.Contains(t, msg.HTMLBody, "10 dakika")
}

// Putting the code in the subject is scored badly by filters and leaks it onto
// lock screens.
func TestLoginCode_SubjectOmitsTheCode(t *testing.T) {
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute)

	assert.NotContains(t, msg.Subject, "482913")
	assert.Equal(t, subjectLoginCode, msg.Subject, "the subject must be a constant, for sender reputation")
}

// Links and images are the two elements that most reliably push a transactional
// message into spam — and a code in spam breaks the only door into the product.
func TestLoginCode_CarriesNoLinksOrImages(t *testing.T) {
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute)

	for _, body := range []string{msg.HTMLBody, msg.TextBody} {
		assert.NotContains(t, body, "<a ")
		assert.NotContains(t, body, "<img")
		assert.NotContains(t, body, "http://")
		assert.NotContains(t, body, "https://")
	}
}

func TestLoginCode_ShortTTLStillReadsAsAtLeastOneMinute(t *testing.T) {
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 20*time.Second)

	assert.Contains(t, msg.TextBody, "1 dakika")
	assert.NotContains(t, msg.TextBody, "0 dakika")
}

func TestNewDevice_DescribesTheMachineAndTime(t *testing.T) {
	at := time.Date(2026, 9, 3, 14, 30, 0, 0, time.UTC)
	msg := NewDevice(model.Address{Email: "user@corp.com"}, "Şube PC", at)

	require.NoError(t, msg.Validate())
	assert.Equal(t, subjectNewDevice, msg.Subject)
	assert.Contains(t, msg.TextBody, "Şube PC")
	assert.Contains(t, msg.TextBody, "03.09.2026 14:30 UTC")
	assert.Contains(t, msg.HTMLBody, "03.09.2026 14:30 UTC")
}

func TestNewDevice_FallsBackWhenNameIsMissing(t *testing.T) {
	msg := NewDevice(model.Address{Email: "user@corp.com"}, "   ", time.Now())

	assert.Contains(t, msg.TextBody, "Bilinmeyen cihaz")
}

// A device name comes from the client. It must never be able to inject markup.
func TestNewDevice_EscapesTheDeviceName(t *testing.T) {
	msg := NewDevice(model.Address{Email: "user@corp.com"}, `<script>alert(1)</script>`, time.Now())

	assert.NotContains(t, msg.HTMLBody, "<script>")
	assert.True(t, strings.Contains(msg.HTMLBody, "&lt;script&gt;"),
		"the name must be HTML-escaped, got: %s", msg.HTMLBody)
}

func TestMessage_ValidateRejectsIncompleteMessages(t *testing.T) {
	full := LoginCode(model.Address{Email: "user@corp.com"}, "482913", time.Minute)

	noRecipient := full
	noRecipient.To = model.Address{}
	assert.ErrorIs(t, noRecipient.Validate(), model.ErrNoRecipient)

	noSubject := full
	noSubject.Subject = ""
	assert.ErrorIs(t, noSubject.Validate(), model.ErrNoSubject)

	noBody := full
	noBody.TextBody, noBody.HTMLBody = "", ""
	assert.ErrorIs(t, noBody.Validate(), model.ErrNoBody)
}
