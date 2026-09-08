package notification

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// LogSender writes messages to the application log instead of delivering them.
// It is the development default: OTP codes and alerts are visible in the server
// output without configuring an external vendor.
type LogSender struct {
	channel string
	log     *slog.Logger
}

func NewLogSender(channel string, log *slog.Logger) *LogSender {
	return &LogSender{channel: channel, log: log}
}

func (s *LogSender) Channel() string { return s.channel }

func (s *LogSender) Send(ctx context.Context, msg Message) error {
	s.log.Info("notification dispatched (log provider)",
		"channel", s.channel, "to", msg.To, "subject", msg.Subject, "body", msg.Body)
	return nil
}

// SMTPSender delivers email over SMTP.
type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
	fromName string
}

func NewSMTPSender(host, port, username, password, from, fromName string) *SMTPSender {
	return &SMTPSender{host: host, port: port, username: username, password: password, from: from, fromName: fromName}
}

func (s *SMTPSender) Channel() string { return ChannelEmail }

func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	addr := net.JoinHostPort(s.host, s.port)
	body := msg.Body
	contentType := "text/plain; charset=UTF-8"
	if msg.HTMLBody != "" {
		body = msg.HTMLBody
		contentType = "text/html; charset=UTF-8"
	}
	headers := []string{
		fmt.Sprintf("From: %s <%s>", s.fromName, s.from),
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
		"Content-Type: " + contentType,
	}
	payload := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, s.from, []string{msg.To}, payload) }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("smtp send: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Second):
		return fmt.Errorf("smtp send timed out")
	}
}
