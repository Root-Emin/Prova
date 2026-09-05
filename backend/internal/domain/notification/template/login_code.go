// Package template builds the transactional messages the auth flow sends.
//
// HTML bodies share a branded card shell with an embedded Prova mark (data URI
// — no public CDN). External http(s) links and tracking pixels stay out, except
// for the optional single magic-link <a> on login codes. The six-digit code
// never appears in the subject line: filters score that pattern poorly, and
// subjects leak into lock-screen notifications.
package template

import (
	"fmt"
	"html"
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

const (
	titleLoginCode  = "Giriş doğrulama kodu"
	titleNewDevice  = "Yeni cihaz eşleştirildi"
	footerLoginCode = "Bu girişi siz talep etmediyseniz bu iletiyi yok sayabilirsiniz. Kodu hiç kimseyle paylaşmayın; Prova ekibi sizden bu kodu asla istemez."
	footerNewDevice = "Bu cihazı siz eşleştirmediyseniz Prova uygulamasındaki cihaz listesinden yetkisini iptal edin."
)

// LoginCode builds the one-time code message.
//
// magicLink boş bırakılabilir. Doluysa aynı iletide hem altı haneli kod hem de
// tek kullanımlık bağlantı yer alır: masaüstünde kodu yazmak, tarayıcıda
// bağlantıya tıklamak kolaydır, ve iki ayrı ileti göndermek kullanıcıya hangi
// iletinin hangi girişe ait olduğunu çözme yükü bindirirdi.
func LoginCode(to model.Address, code string, ttl time.Duration, magicLink string) model.Message {
	minutes := int(ttl.Round(time.Minute).Minutes())
	if minutes < 1 {
		minutes = 1
	}

	var validity string
	if magicLink != "" {
		validity = fmt.Sprintf("Kod ve bağlantı %d dakika geçerlidir ve yalnızca bir kez kullanılabilir.", minutes)
	} else {
		validity = fmt.Sprintf("Kod %d dakika geçerlidir ve yalnızca bir kez kullanılabilir.", minutes)
	}

	htmlBody := renderShell(
		titleLoginCode,
		footerLoginCode,
		bodyHTML(
			bodyParagraph("Prova'ya giriş yapmak için doğrulama kodunuz:"),
			otpPanel(code),
			bodyParagraph(validity),
		),
		magicLink,
	)

	lines := []string{
		"Prova'ya giriş yapmak için doğrulama kodunuz:",
		"",
		code,
		"",
	}
	if magicLink != "" {
		lines = append(lines,
			"Kodu yazmak istemiyorsanız doğrudan bu bağlantıyla giriş yapabilirsiniz:",
			magicLink,
			"",
			fmt.Sprintf("Kod ve bağlantı %d dakika geçerlidir ve yalnızca bir kez kullanılabilir.", minutes),
		)
	} else {
		lines = append(lines,
			fmt.Sprintf("Kod %d dakika geçerlidir ve yalnızca bir kez kullanılabilir.", minutes),
		)
	}
	lines = append(lines,
		"",
		"Bu girişi siz talep etmediyseniz bu iletiyi yok sayabilirsiniz.",
		"Kodu hiç kimseyle paylaşmayın; Prova ekibi sizden bu kodu asla istemez.",
	)

	return model.Message{
		To:       to,
		Subject:  subjectLoginCode,
		TextBody: strings.Join(lines, "\n"),
		HTMLBody: htmlBody,
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

	detail := template.HTML(
		`<p style="margin:0 0 8px;font-family:` + fontBody + `;font-size:15px;line-height:1.6;color:` + colorInk + `;"><strong>Cihaz:</strong> ` +
			html.EscapeString(deviceName) + `</p>` +
			`<p style="margin:0 0 16px;font-family:` + fontBody + `;font-size:15px;line-height:1.6;color:` + colorInk + `;"><strong>Tarih:</strong> ` +
			html.EscapeString(stamp) + `</p>`,
	)

	htmlBody := renderShell(
		titleNewDevice,
		footerNewDevice,
		bodyHTML(
			bodyParagraph("Prova hesabınıza yeni bir cihaz eşleştirildi."),
			detail,
		),
		"",
	)

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
		HTMLBody: htmlBody,
		Tags:     map[string]string{"category": "new_device"},
	}
}
