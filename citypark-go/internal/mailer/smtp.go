package mailer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/config"
)

var ErrDelivery = errors.New("email delivery failed")

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}
type Dialer func(ctx context.Context, network, address string) (net.Conn, error)
type SMTP struct {
	settings config.SMTP
	dial     Dialer
	roots    *x509.CertPool
}

func NewSMTP(settings config.SMTP, dial Dialer) *SMTP {
	if dial == nil {
		d := &net.Dialer{Timeout: 10 * time.Second}
		dial = d.DialContext
	}
	return &SMTP{settings: settings, dial: dial}
}

func (s *SMTP) Send(ctx context.Context, to, subject, body string) error {
	return s.send(ctx, to, subject, "text/plain; charset=UTF-8", body)
}

// SendAlternative sends equivalent plain-text and HTML versions of a message.
func (s *SMTP) SendAlternative(ctx context.Context, to, subject, plainText, html string) error {
	if invalidHeader(subject) {
		return ErrDelivery
	}
	var body strings.Builder
	parts := multipart.NewWriter(&body)
	if err := writeAlternative(parts, "text/plain", plainText); err != nil {
		return ErrDelivery
	}
	if err := writeAlternative(parts, "text/html", html); err != nil {
		return ErrDelivery
	}
	if err := parts.Close(); err != nil {
		return ErrDelivery
	}
	contentType := mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": parts.Boundary()})
	return s.send(ctx, to, mime.BEncoding.Encode("UTF-8", subject), contentType, body.String())
}

func writeAlternative(parts *multipart.Writer, contentType, body string) error {
	part, err := parts.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {contentType + "; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return err
	}
	encoded := quotedprintable.NewWriter(part)
	if _, err := io.WriteString(encoded, body); err != nil {
		return err
	}
	return encoded.Close()
}

func (s *SMTP) send(ctx context.Context, to, subject, contentType, body string) error {
	if !s.settings.Enabled || invalidHeader(to) || invalidHeader(subject) || invalidHeader(s.settings.From) || subject == "" {
		return ErrDelivery
	}
	addr, err := mail.ParseAddress(to)
	if err != nil || addr.Address != to {
		return ErrDelivery
	}
	endpoint := net.JoinHostPort(s.settings.Host, strconv.Itoa(s.settings.Port))
	conn, err := s.dial(ctx, "tcp", endpoint)
	if err != nil {
		return ErrDelivery
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return ErrDelivery
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	tlsConfig := &tls.Config{ServerName: s.settings.Host, MinVersion: tls.VersionTLS12, RootCAs: s.roots}
	if s.settings.TLSMode == "tls" {
		secured := tls.Client(conn, tlsConfig)
		if err := secured.HandshakeContext(ctx); err != nil {
			return ErrDelivery
		}
		conn = secured
	}
	client, err := smtp.NewClient(conn, s.settings.Host)
	if err != nil {
		return ErrDelivery
	}
	defer client.Close()
	if s.settings.TLSMode == "starttls" {
		ok, _ := client.Extension("STARTTLS")
		if !ok || client.StartTLS(tlsConfig) != nil {
			return ErrDelivery
		}
	}
	if _, ok := client.TLSConnectionState(); !ok {
		return ErrDelivery
	}
	if s.settings.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.settings.Username, s.settings.Password, s.settings.Host)); err != nil {
			return ErrDelivery
		}
	}
	if err := client.Mail(s.settings.From); err != nil {
		return ErrDelivery
	}
	if err := client.Rcpt(to); err != nil {
		return ErrDelivery
	}
	writer, err := client.Data()
	if err != nil {
		return ErrDelivery
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s\r\n\r\n%s", s.settings.From, to, subject, contentType, body)
	if _, err := io.WriteString(writer, message); err != nil {
		_ = writer.Close()
		return ErrDelivery
	}
	if err := writer.Close(); err != nil {
		return ErrDelivery
	}
	if err := client.Quit(); err != nil {
		return ErrDelivery
	}
	return nil
}
func invalidHeader(v string) bool { return strings.ContainsAny(v, "\r\n") }
