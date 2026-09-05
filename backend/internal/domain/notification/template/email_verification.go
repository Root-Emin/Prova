package template

import (
	"fmt"
	"strings"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
)

const (
	subjectEmailVerification = "Prova e-posta doğrulama kodunuz"
	titleEmailVerification   = "E-posta doğrulama kodu"
	footerEmailVerification  = "Bu hesabı siz oluşturmadıysanız iletiyi yok sayabilirsiniz. Kodu hiç kimseyle paylaşmayın; Prova ekibi sizden bu kodu asla istemez."
)

// EmailVerification builds an ownership-verification message. It intentionally
// contains no login or magic-link action: redeeming this code cannot create a
// session.
func EmailVerification(to model.Address, code string, ttl time.Duration) model.Message {
	minutes := int(ttl.Round(time.Minute).Minutes())
	if minutes < 1 {
		minutes = 1
	}
	validity := fmt.Sprintf("Kod %d dakika geçerlidir ve yalnızca bir kez kullanılabilir.", minutes)

	htmlBody := renderShell(
		titleEmailVerification,
		footerEmailVerification,
		bodyHTML(
			bodyParagraph("E-posta adresinizi doğrulamak için kodunuz:"),
			otpPanel(code),
			bodyParagraph(validity),
		),
		"",
	)

	return model.Message{
		To:      to,
		Subject: subjectEmailVerification,
		TextBody: strings.Join([]string{
			"E-posta adresinizi doğrulamak için kodunuz:",
			"",
			code,
			"",
			validity,
			"",
			"Bu hesabı siz oluşturmadıysanız iletiyi yok sayabilirsiniz.",
			"Kodu hiç kimseyle paylaşmayın; Prova ekibi sizden bu kodu asla istemez.",
		}, "\n"),
		HTMLBody: htmlBody,
		Tags:     map[string]string{"category": "email_verification"},
	}
}
