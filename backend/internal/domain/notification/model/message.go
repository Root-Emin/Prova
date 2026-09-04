package model

import "strings"

// Address is an e-mail recipient or sender.
type Address struct {
	Email string
	Name  string
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
