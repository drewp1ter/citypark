package telegram

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProxyHTTPClient(t *testing.T) {
	original := http.DefaultTransport.(*http.Transport).Proxy
	for _, scheme := range []string{"socks5", "socks5h"} {
		t.Run(scheme, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			result := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					result <- err
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				reader := bufio.NewReader(conn)
				check := func(want string) bool {
					b := make([]byte, len(want))
					_, err := io.ReadFull(reader, b)
					if err != nil {
						result <- err
						return false
					}
					if string(b) != want {
						result <- &protocolError{}
						return false
					}
					return true
				}
				if !check("\x05\x02\x00\x02") {
					return
				}
				conn.Write([]byte{5, 2})
				if !check("\x01\x04user\x08password") {
					return
				}
				conn.Write([]byte{1, 0})
				if !check("\x05\x01\x00\x03\x10telegram.invalid") {
					return
				}
				port := make([]byte, 2)
				if _, err := io.ReadFull(reader, port); err != nil {
					result <- err
					return
				}
				if binary.BigEndian.Uint16(port) != 80 {
					result <- &protocolError{}
					return
				}
				conn.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 80})
				req, err := http.ReadRequest(reader)
				if err != nil {
					result <- err
					return
				}
				defer req.Body.Close()
				_, err = io.Copy(io.Discard, req.Body)
				if err != nil {
					result <- err
					return
				}
				if req.Host != "telegram.invalid" || req.URL.Path != "/bot"+testToken+"/sendMessage" {
					result <- &protocolError{}
					return
				}
				_, err = io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 11\r\nConnection: close\r\n\r\n{\"ok\":true}")
				result <- err
			}()
			proxyURL, _ := url.Parse(scheme + "://user:password@" + listener.Addr().String())
			client := NewProxyHTTPClient(proxyURL)
			defer client.CloseIdleConnections()
			err = NewClient("http://telegram.invalid", testToken, client).SendMessage(context.Background(), "1", "hello")
			if err != nil {
				t.Fatal(err)
			}
			if err := <-result; err != nil {
				t.Fatal(err)
			}
		})
	}
	if original == nil || http.DefaultTransport.(*http.Transport).Proxy == nil {
		t.Fatal("default transport changed")
	}
}

type protocolError struct{}

func (*protocolError) Error() string { return "unexpected SOCKS or HTTP request" }

func TestProxyFailureDoesNotSendDirect(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	proxyURL, _ := url.Parse("socks5://user:password@" + address)
	client := NewProxyHTTPClient(proxyURL)
	defer client.CloseIdleConnections()
	err = NewClient(target.URL, testToken, client).SendMessage(context.Background(), "1", "hello")
	if err == nil || called {
		t.Fatalf("expected proxy failure without direct fallback: %v", err)
	}
	if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), testToken) {
		t.Fatal("error leaked credentials")
	}
}

func TestProxyHTTPClientEmpty(t *testing.T) {
	client := NewProxyHTTPClient(nil)
	if client.Timeout != 15*time.Second {
		t.Fatal("wrong timeout")
	}
	if client.Transport == http.DefaultTransport {
		t.Fatal("transport is shared")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"ok":true}`) }))
	defer server.Close()
	defer client.CloseIdleConnections()
	if err := NewClient(server.URL, testToken, client).SendMessage(context.Background(), "1", "hello"); err != nil {
		t.Fatal(err)
	}
}
