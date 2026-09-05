package model

import (
	"fmt"
	"net/mail"
	"strings"
)

// Address is an e-mail recipient or sender.
type Address struct {
	Email string
	Name  string
}

// ParseAddress accepts either a bare address or an RFC 5322 display address.
// Configuration uses the latter for Resend (for example, "Prova
// <onboarding@resend.dev>"). Keeping parsing here prevents adapters from
// accidentally producing a malformed From header or duplicating a display
// name that is already present in the configured value.
func ParseAddress(raw, defaultName string) (Address, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\r\n") {
		return Address{}, fmt.Errorf("invalid e-mail address")
	}

	parsed, err := mail.ParseAddress(raw)
	if err != nil || strings.TrimSpace(parsed.Address) == "" {
		return Address{}, fmt.Errorf("invalid e-mail address")
	}
	name := strings.TrimSpace(parsed.Name)
	if name == "" {
		name = strings.TrimSpace(defaultName)
	}
	return Address{Email: parsed.Address, Name: name}, nil
}

// String renders the address in RFC 5322 form ("Name <user@example.com>").
func (a Address) String() string {
	if a.Name == "" {
		return a.Email
	}
	return a.Name + " <" + a.Email + ">"
}

// Message is a transactional e-mail, expressed in terms no provider knows about.
//
// Both TextBody and HTMLBody are always populated. Codes are delivered to
// mailboxes we do not control, and a plain-text alternative is what keeps the
// message readable in clients that refuse HTML — the login flow has no fallback
// if the message is unreadable.
type Message struct {
	To       Address
	Subject  string
	TextBody string
	HTMLBody string

	// Tags are provider-neutral labels for delivery analytics (for example
	// "category=login_code"). Adapters drop them if the provider has no
	// equivalent.
	Tags map[string]string
}

// Validate reports whether the message can be handed to a sender.
func (m Message) Validate() error {
	switch {
	case strings.TrimSpace(m.To.Email) == "":
		return ErrNoRecipient
	case strings.TrimSpace(m.Subject) == "":
		return ErrNoSubject
	case strings.TrimSpace(m.TextBody) == "" && strings.TrimSpace(m.HTMLBody) == "":
		return ErrNoBody
	}
	return nil
}
