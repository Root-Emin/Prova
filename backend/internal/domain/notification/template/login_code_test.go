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
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute, "")

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
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute, "")

	assert.NotContains(t, msg.Subject, "482913")
	assert.Equal(t, subjectLoginCode, msg.Subject, "the subject must be a constant, for sender reputation")
}

// Without a magic link the message must stay free of external http(s) URLs and
// of <a> tags. The embedded Prova mark (data:image/png) is the sole intentional
// image — no tracking pixels, no CDN.
func TestLoginCode_CarriesNoLinksOrImagesWithoutMagicLink(t *testing.T) {
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute, "")

	assert.NotContains(t, msg.HTMLBody, "<a ")
	assert.NotContains(t, msg.TextBody, "<a ")
	assert.NotContains(t, msg.HTMLBody, "http://")
	assert.NotContains(t, msg.HTMLBody, "https://")
	assert.NotContains(t, msg.TextBody, "http://")
	assert.NotContains(t, msg.TextBody, "https://")

	assert.Contains(t, msg.HTMLBody, "data:image/png")
	assert.Contains(t, msg.HTMLBody, `alt="Prova"`)
	assert.Contains(t, msg.HTMLBody, "<img")
	assert.Contains(t, msg.HTMLBody, "Prova")
	assert.Contains(t, msg.HTMLBody, brandTagline)
}

func TestLoginCode_ShortTTLStillReadsAsAtLeastOneMinute(t *testing.T) {
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 20*time.Second, "")

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
	assert.Contains(t, msg.HTMLBody, "data:image/png")
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
	full := LoginCode(model.Address{Email: "user@corp.com"}, "482913", time.Minute, "")

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

// Aynı ileti hem kodu hem bağlantıyı taşımak zorunda: kullanıcı hangisini
// isterse onu kullanır, ve iki ayrı ileti hangisinin hangi girişe ait olduğunu
// çözme yükünü kullanıcıya bindirirdi.
func TestLoginCode_CarriesBothCodeAndMagicLink(t *testing.T) {
	link := "https://app.example.com/auth/magic?token=abc123"
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute, link)

	if !strings.Contains(msg.TextBody, "482913") || !strings.Contains(msg.TextBody, link) {
		t.Fatalf("düz metin gövdesi hem kodu hem bağlantıyı içermeli:\n%s", msg.TextBody)
	}
	if !strings.Contains(msg.HTMLBody, "482913") || !strings.Contains(msg.HTMLBody, link) {
		t.Fatalf("HTML gövdesi hem kodu hem bağlantıyı içermeli")
	}
	if strings.Count(msg.HTMLBody, "<a ") != 1 {
		t.Fatalf("magic link varken tek bir <a> olmalı, got %d", strings.Count(msg.HTMLBody, "<a "))
	}
}

// Bağlantı üretilemediğinde ileti yine gitmeli; bağlantı bir kolaylıktır,
// girişin tek yolu değil.
func TestLoginCode_OmitsLinkSectionWhenLinkIsEmpty(t *testing.T) {
	msg := LoginCode(model.Address{Email: "user@corp.com"}, "482913", 10*time.Minute, "")

	if strings.Contains(msg.TextBody, "bağlantıyla giriş") {
		t.Fatalf("bağlantı yokken bağlantı metni yazılmamalı:\n%s", msg.TextBody)
	}
	if !strings.Contains(msg.TextBody, "482913") {
		t.Fatalf("kod her zaman yer almalı")
	}
}
