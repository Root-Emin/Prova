// Package template builds the transactional messages the auth flow sends.
//
// The templates are deliberately plain: no links, no images, no tracking
// pixels. A six-digit code that lands in a spam folder breaks the only door
// into the product, so every element that raises a spam score is left out. The
// code never appears in the subject line either — filters score that pattern
// poorly, and subjects leak into lock-screen notifications.
package template

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
)

// Subjects are fixed strings. Predictable subjects build a stable sending
// reputation, and stable reputation is what keeps the code out of spam.
const (
	subjectLoginCode = "Prova giriş doğrulama kodunuz"
	subjectNewDevice = "Prova hesabınıza yeni bir cihaz eklendi"
)

var loginCodeHTML = template.Must(template.New("login_code").Parse(`<!doctype html>
<html lang="tr">
<body style="margin:0;padding:24px;background:#ffffff;font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;color:#111111;">
<p style="font-size:15px;line-height:1.6;margin:0 0 20px;">Prova'ya giriş yapmak için doğrulama kodunuz:</p>
<p style="font-size:34px;font-weight:700;letter-spacing:6px;margin:0 0 20px;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;">{{.Code}}</p>
<p style="font-size:15px;line-height:1.6;margin:0 0 20px;">Kod {{.Minutes}} dakika geçerlidir ve yalnızca bir kez kullanılabilir.</p>
<p style="font-size:15px;line-height:1.6;margin:0;color:#555555;">Bu girişi siz talep etmediyseniz bu iletiyi yok sayabilirsiniz. Kodu hiç kimseyle paylaşmayın; Prova ekibi sizden bu kodu asla istemez.</p>
</body>
</html>`))

var newDeviceHTML = template.Must(template.New("new_device").Parse(`<!doctype html>
<html lang="tr">
<body style="margin:0;padding:24px;background:#ffffff;font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;color:#111111;">
<p style="font-size:15px;line-height:1.6;margin:0 0 20px;">Prova hesabınıza yeni bir cihaz eşleştirildi.</p>
<p style="font-size:15px;line-height:1.6;margin:0 0 8px;"><strong>Cihaz:</strong> {{.DeviceName}}</p>
<p style="font-size:15px;line-height:1.6;margin:0 0 20px;"><strong>Tarih:</strong> {{.OccurredAt}}</p>
<p style="font-size:15px;line-height:1.6;margin:0;color:#555555;">Bu cihazı siz eşleştirmediyseniz Prova uygulamasındaki cihaz listesinden yetkisini iptal edin.</p>
</body>
</html>`))

// LoginCode builds the one-time code message.
func LoginCode(to model.Address, code string, ttl time.Duration) model.Message {
	minutes := int(ttl.Round(time.Minute).Minutes())
	if minutes < 1 {
		minutes = 1
	}

	data := struct {
		Code    string
		Minutes int
	}{Code: code, Minutes: minutes}

	text := strings.Join([]string{
		"Prova'ya giriş yapmak için doğrulama kodunuz:",
		"",
		code,
		"",
		fmt.Sprintf("Kod %d dakika geçerlidir ve yalnızca bir kez kullanılabilir.", minutes),
		"",
		"Bu girişi siz talep etmediyseniz bu iletiyi yok sayabilirsiniz.",
		"Kodu hiç kimseyle paylaşmayın; Prova ekibi sizden bu kodu asla istemez.",
	}, "\n")

	return model.Message{
		To:       to,
		Subject:  subjectLoginCode,
		TextBody: text,
		HTMLBody: render(loginCodeHTML, data),
		Tags:     map[string]string{"category": "login_code"},
	}
}

// NewDevice builds the notification sent when a device is paired with an
// account for the first time. Pairing happens silently at login, so this
// message is the user's only chance to notice a mailbox takeover.
func NewDevice(to model.Address, deviceName string, occurredAt time.Time) model.Message {
	if strings.TrimSpace(deviceName) == "" {
		deviceName = "Bilinmeyen cihaz"
	}
	stamp := occurredAt.UTC().Format("02.01.2006 15:04 UTC")

	data := struct {
		DeviceName string
		OccurredAt string
	}{DeviceName: deviceName, OccurredAt: stamp}

	text := strings.Join([]string{
		"Prova hesabınıza yeni bir cihaz eşleştirildi.",
		"",
		"Cihaz: " + deviceName,
		"Tarih: " + stamp,
		"",
		"Bu cihazı siz eşleştirmediyseniz Prova uygulamasındaki cihaz listesinden yetkisini iptal edin.",
	}, "\n")

	return model.Message{
		To:       to,
		Subject:  subjectNewDevice,
		TextBody: text,
		HTMLBody: render(newDeviceHTML, data),
		Tags:     map[string]string{"category": "new_device"},
	}
}

// render executes a parsed template. The templates are compile-time constants
// with escaped data, so execution cannot fail for reasons a caller could act
// on; an empty HTML body still leaves the text alternative intact.
func render(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}
