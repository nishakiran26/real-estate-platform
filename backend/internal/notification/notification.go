// Package notification defines provider-agnostic delivery channels. Services
// depend on the interfaces only, so an SMS or WhatsApp vendor can be swapped by
// registering a different implementation in main.go.
package notification

import "context"

// Message is a single outbound notification on any channel.
type Message struct {
	To       string            // email address or E.164 mobile number
	Subject  string            // used by email; ignored by SMS
	Body     string            // plain-text body
	HTMLBody string            // optional HTML body for email
	Template string            // optional provider-side template identifier
	Data     map[string]string // template variables
}

// Sender delivers a message on one channel.
type Sender interface {
	Send(ctx context.Context, msg Message) error
	Channel() string
}

// Dispatcher routes a message to the sender registered for a channel.
type Dispatcher struct {
	senders map[string]Sender
}

func NewDispatcher(senders ...Sender) *Dispatcher {
	d := &Dispatcher{senders: make(map[string]Sender, len(senders))}
	for _, s := range senders {
		if s != nil {
			d.senders[s.Channel()] = s
		}
	}
	return d
}

// Channels available for delivery.
const (
	ChannelEmail    = "EMAIL"
	ChannelSMS      = "SMS"
	ChannelWhatsApp = "WHATSAPP"
)

// Send delivers on the requested channel. An unregistered channel is a no-op so
// that a missing vendor configuration never breaks a business flow.
func (d *Dispatcher) Send(ctx context.Context, channel string, msg Message) error {
	s, ok := d.senders[channel]
	if !ok {
		return nil
	}
	return s.Send(ctx, msg)
}

// Has reports whether a channel has a registered sender.
func (d *Dispatcher) Has(channel string) bool {
	_, ok := d.senders[channel]
	return ok
}
