// Package smtp, SMTP üzerinden teslim eden gönderici adaptörüdür.
//
// Geliştirmede Mailpit'e, gerekirse üretimde bir SMTP rölesine bağlanır.
// Resend adaptörünün yanında durur; ikisi de notification.Sender portunu
// uygular ve uygulama katmanı hangisinin arkada olduğunu bilmez.
package smtp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"time"

	notifyModel "github.com/masterfabric-go/masterfabric/internal/domain/notification/model"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

// Sender, SMTP teslim adaptörü.
type Sender struct {
	cfg  config.SMTPConfig
	from notifyModel.Address
	// replyTo boş bırakılabilir.
	replyTo string
	timeout time.Duration
}

// New builds the sender from configuration.
func New(cfg config.EmailConfig) (*Sender, error) {
	if strings.TrimSpace(cfg.SMTP.Host) == "" {
		return nil, errors.New("smtp: SMTP_HOST boş")
	}
	if strings.TrimSpace(cfg.FromAddress) == "" {
		return nil, errors.New("smtp: EMAIL_FROM_ADDRESS boş")
	}
	from, err := notifyModel.ParseAddress(cfg.FromAddress, cfg.FromName)
	if err != nil {
		return nil, fmt.Errorf("smtp: geçersiz gönderen adresi: %w", err)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Sender{
		cfg:     cfg.SMTP,
		from:    from,
		replyTo: cfg.ReplyTo,
		timeout: timeout,
	}, nil
}

// Name implements service.Sender.
func (s *Sender) Name() string { return config.ProviderSMTP }

// Send implements service.Sender.
func (s *Sender) Send(ctx context.Context, msg notifyModel.Message) (string, error) {
	if err := msg.Validate(); err != nil {
		return "", err
	}

	messageID := fmt.Sprintf("<%d.%s@prova>", time.Now().UnixNano(), sanitizeDomain(s.from.Email))
	payload := s.compose(msg, messageID)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	client, err := s.dial(ctx, addr)
	if err != nil {
		return "", err
	}
	defer func() { _ = client.Quit() }()

	if err := client.Mail(s.from.Email); err != nil {
		return "", fmt.Errorf("smtp: gönderen reddedildi: %w", err)
	}
	if err := client.Rcpt(msg.To.Email); err != nil {
		return "", fmt.Errorf("smtp: alıcı reddedildi: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("smtp: veri akışı açılamadı: %w", err)
	}
	if _, err := writer.Write([]byte(payload)); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("smtp: ileti yazılamadı: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("smtp: ileti tamamlanamadı: %w", err)
	}

	return messageID, nil
}

// dial, sunucuya bağlanır ve gerekiyorsa STARTTLS'e geçer.
func (s *Sender) dial(ctx context.Context, addr string) (*smtp.Client, error) {
	dialer := &tls.Dialer{}
	_ = dialer
	// net/smtp'nin kendi Dial'ı bağlam almıyor; zaman aşımı bağlantı
	// üzerinden uygulanıyor. Mailpit yerelde anında cevap verir, üretimde
	// röle yavaşsa istek burada takılmamalı.
	conn, err := (&netDialer{timeout: s.timeout}).dial(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("smtp: bağlanılamadı: %w", err)
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("smtp: istemci kurulamadı: %w", err)
	}

	if s.cfg.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, errors.New("smtp: SMTP_USE_TLS açık ama sunucu STARTTLS desteklemiyor")
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("smtp: STARTTLS başarısız: %w", err)
		}
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("smtp: kimlik doğrulama başarısız: %w", err)
		}
	}

	return client, nil
}

// compose, çok parçalı (metin + HTML) iletiyi kurar.
//
// Her iki gövde de gönderiliyor: kod, kontrol etmediğimiz posta kutularına
// düşüyor ve HTML'i reddeden bir istemcide okunamayan ileti girişin tek
// kapısını kapatır.
func (s *Sender) compose(msg notifyModel.Message, messageID string) string {
	boundary := fmt.Sprintf("prova-%d", time.Now().UnixNano())

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", encodeAddress(s.from))
	fmt.Fprintf(&b, "To: %s\r\n", encodeAddress(msg.To))
	if s.replyTo != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", s.replyTo)
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", msg.Subject))
	fmt.Fprintf(&b, "Message-ID: %s\r\n", messageID)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
	for key, value := range msg.Tags {
		// Etiketler sağlayıcı-nötr başlık olarak taşınıyor; SMTP'nin
		// etiket kavramı yok ama teslim analizinde okunabilir kalıyorlar.
		fmt.Fprintf(&b, "X-Prova-%s: %s\r\n", strings.ToUpper(key[:1])+key[1:], value)
	}
	b.WriteString("\r\n")

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(msg.TextBody)
	b.WriteString("\r\n")

	if msg.HTMLBody != "" {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(msg.HTMLBody)
		b.WriteString("\r\n")
	}

	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.String()
}

func encodeAddress(a notifyModel.Address) string {
	if a.Name == "" {
		return a.Email
	}
	return fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("UTF-8", a.Name), a.Email)
}

func sanitizeDomain(email string) string {
	_, domain, found := strings.Cut(email, "@")
	if !found || domain == "" {
		return "local"
	}
	return domain
}
