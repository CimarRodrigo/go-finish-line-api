// Package email is shared infrastructure for sending transactional email.
// It has no domain: sending a message is a "how", not a business "what".
package email

import "context"

// Message is a transactional email ready to send.
type Message struct {
	To      string
	Subject string
	HTML    string
}

// Sender delivers a Message through some provider. Modules depend on this
// interface, not on a concrete provider.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
