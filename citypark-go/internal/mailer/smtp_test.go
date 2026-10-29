package mailer

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/config"
)

func testCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"localhost"}, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certPEM)
	return cert, roots
}

type smtpFake struct {
	listener  net.Listener
	mu        sync.Mutex
	commands  []string
	message   string
	fail      string
	startTLS  bool
	offerTLS  bool
	tlsConfig *tls.Config
}

func newFake(t *testing.T, mode string, offer bool, fail string) (*smtpFake, *x509.CertPool) {
	t.Helper()
	cert, roots := testCertificate(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &smtpFake{listener: ln, startTLS: mode == "starttls", offerTLS: offer, fail: fail, tlsConfig: &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}}
	if mode == "tls" {
		f.listener = tls.NewListener(ln, f.tlsConfig)
	}
	t.Cleanup(func() { f.listener.Close() })
	go func() {
		for {
			conn, err := f.listener.Accept()
			if err != nil {
				return
			}
			go f.serve(conn)
		}
	}()
	return f, roots
}
func (f *smtpFake) serve(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	write := func(s string) bool { _, err := fmt.Fprint(conn, s+"\r\n"); return err == nil }
	if !write("220 localhost ready") {
		return
	}
	secured := !f.startTLS
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)
		f.mu.Lock()
		f.commands = append(f.commands, line)
		f.mu.Unlock()
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			if f.startTLS && f.offerTLS && !secured {
				if !write("250-localhost") || !write("250-STARTTLS") || !write("250 AUTH PLAIN") {
					return
				}
			} else {
				if !write("250-localhost") || !write("250 AUTH PLAIN") {
					return
				}
			}
		case upper == "STARTTLS":
			if !f.offerTLS || !write("220 start tls") {
				return
			}
			tlsConn := tls.Server(conn, f.tlsConfig)
			if tlsConn.Handshake() != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
			secured = true
		case strings.HasPrefix(upper, "AUTH"):
			if !secured {
				write("530 require tls")
				return
			}
			if f.fail == "auth" {
				write("535 rejected")
			} else {
				write("235 ok")
			}
		case strings.HasPrefix(upper, "MAIL FROM"):
			if f.fail == "mail" {
				write("550 rejected")
			} else {
				write("250 ok")
			}
		case strings.HasPrefix(upper, "RCPT TO"):
			if f.fail == "recipient" {
				write("550 rejected")
			} else {
				write("250 ok")
			}
		case upper == "DATA":
			if !write("354 go") {
				return
			}
			var body strings.Builder
			for {
				part, e := reader.ReadString('\n')
				if e != nil {
					return
				}
				if strings.TrimSpace(part) == "." {
					break
				}
				body.WriteString(part)
			}
			f.mu.Lock()
			f.message = body.String()
			f.mu.Unlock()
			write("250 queued")
		case upper == "QUIT":
			write("221 bye")
			return
		default:
			write("500 unknown")
		}
	}
}
func (f *smtpFake) commandsCopy() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.commands...)
}

func TestSMTPSend(t *testing.T) {
	for _, mode := range []string{"tls", "starttls"} {
		t.Run(mode, func(t *testing.T) {
			f, roots := newFake(t, mode, true, "")
			port := f.listener.Addr().(*net.TCPAddr).Port
			s := NewSMTP(config.SMTP{Host: "localhost", Port: port, Username: "user", Password: "secret", From: "from@example.com", TLSMode: mode, Enabled: true}, nil)
			s.roots = roots
			if err := s.Send(context.Background(), "to@example.com", "Hello", "message body"); err != nil {
				t.Fatal(err)
			}
			f.mu.Lock()
			message := f.message
			f.mu.Unlock()
			if !strings.Contains(message, "Subject: Hello") || !strings.Contains(message, "message body") {
				t.Fatal(message)
			}
			commands := f.commandsCopy()
			start, auth := -1, -1
			for i, c := range commands {
				if c == "STARTTLS" {
					start = i
				}
				if strings.HasPrefix(c, "AUTH") {
					auth = i
				}
			}
			if auth < 0 || mode == "starttls" && (start < 0 || auth < start) {
				t.Fatalf("commands: %v", commands)
			}
		})
	}
	for _, tc := range []struct {
		name, mode, fail string
		offer            bool
		trusted          bool
	}{
		{"no starttls", "starttls", "", false, true}, {"bad certificate", "tls", "", true, false}, {"auth refusal", "tls", "auth", true, true}, {"recipient refusal", "tls", "recipient", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, roots := newFake(t, tc.mode, tc.offer, tc.fail)
			s := NewSMTP(config.SMTP{Host: "localhost", Port: f.listener.Addr().(*net.TCPAddr).Port, Username: "user", Password: "secret", From: "from@example.com", TLSMode: tc.mode, Enabled: true}, nil)
			if tc.trusted {
				s.roots = roots
			}
			err := s.Send(context.Background(), "to@example.com", "Hello", "secret body")
			if err != ErrDelivery {
				t.Fatalf("error: %v", err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal(err)
			}
			if !tc.offer {
				for _, cmd := range f.commandsCopy() {
					if strings.HasPrefix(cmd, "AUTH") {
						t.Fatal("credentials sent before TLS")
					}
				}
			}
		})
	}
	f, roots := newFake(t, "tls", true, "")
	s := NewSMTP(config.SMTP{Host: "localhost", Port: f.listener.Addr().(*net.TCPAddr).Port, Username: "user", Password: "secret", From: "from@example.com", TLSMode: "tls", Enabled: true}, nil)
	s.roots = roots
	for _, tc := range []struct{ to, subject string }{{"a@example.com\r\nBcc: victim@example.com", "Hello"}, {"to@example.com", "Hello\nBcc: victim@example.com"}} {
		if err := s.Send(context.Background(), tc.to, tc.subject, "body"); err != ErrDelivery {
			t.Fatalf("header injection: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Send(ctx, "to@example.com", "Hello", "body"); err != ErrDelivery {
		t.Fatalf("cancel: %v", err)
	}
	stalled := NewSMTP(config.SMTP{Host: "localhost", Port: 25, Username: "user", Password: "secret", From: "from@example.com", TLSMode: "starttls", Enabled: true}, func(context.Context, string, string) (net.Conn, error) {
		client, server := net.Pipe()
		t.Cleanup(func() { server.Close() })
		return client, nil
	})
	timeout, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	if err := stalled.Send(timeout, "to@example.com", "Hello", "body"); err != ErrDelivery {
		t.Fatalf("timeout: %v", err)
	}
}
