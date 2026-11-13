// Package fakeglobal is a conformance fixture: a fake keeps the messages it
// was sent in a package-level variable.
package fakeglobal

// Mailer is the port for sending messages.
type Mailer interface {
	Send(message string)
}

var sent []string

// FakeMailer is the fake of Mailer.
type FakeMailer struct{}

// Send keeps message.
func (FakeMailer) Send(message string) { sent = append(sent, message) }

// Sent returns the messages sent.
func (FakeMailer) Sent() []string { return sent }
