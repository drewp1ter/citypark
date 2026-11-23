package google

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type yandexTransport func(*http.Request) (*http.Response, error)

func (f yandexTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestYandexExchange(t *testing.T) {
	for _, appID := range []string{"citypark", "other"} {
		t.Run(appID, func(t *testing.T) {
			s := NewYandexService(nil, nil, "citypark", "secret", "https://citypark.example/account/yandex/callback")
			s.httpClient.Transport = yandexTransport(func(r *http.Request) (*http.Response, error) {
				var body string
				switch r.URL.String() {
				case "https://oauth.yandex.ru/token":
					if err := r.ParseForm(); err != nil || r.Method != http.MethodPost || r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code_verifier") != "verifier" || r.Form.Get("code") != "code" || r.Form.Get("client_secret") != "secret" {
						t.Fatal("invalid authorization-code exchange")
					}
					body = `{"access_token":"access"}`
				case yandexProfileURL:
					if r.Method != http.MethodGet || r.Header.Get("Authorization") != "OAuth access" {
						t.Fatal("invalid Yandex profile request")
					}
					body = `{"id":"123","client_id":"` + appID + `","real_name":"Customer"}`
				default:
					t.Fatal("unexpected Yandex endpoint")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			got, err := s.exchangeYandex(context.Background(), "code", "verifier")
			if appID != "citypark" {
				if err == nil {
					t.Fatal("accepted identity for another application")
				}
			} else if err != nil || got.Subject != "123" || got.Name != "Customer" {
				t.Fatalf("profile = %+v, err = %v", got, err)
			}
		})
	}
}

func TestYandexCustomerLoginAndLink(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := clock.Fixed(time.Now().UTC())
	s := NewYandexService(db, c, "citypark", "secret", "https://citypark.example/account/yandex/callback")
	subject := "123"
	s.httpClient.Transport = yandexTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"access_token":"access"}`
		if r.URL.String() == yandexProfileURL {
			body = `{"id":"` + subject + `","client_id":"citypark"}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	run := func(intent string, session *customer.Session) (CompletionResult, error) {
		t.Helper()
		start, err := s.Start(ctx, intent, session)
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(start.AuthorizationURL)
		if err != nil {
			t.Fatal(err)
		}
		return s.Complete(ctx, u.Query().Get("state"), start.FlowToken, "code", "", session)
	}
	first, err := run("login", nil)
	if err != nil || first.User == nil || first.User.Email != "" || first.Token == "" {
		t.Fatalf("first login failed: %v", err)
	}
	second, err := run("login", nil)
	if err != nil || second.User == nil || second.User.ID != first.User.ID {
		t.Fatalf("repeat login failed: %v", err)
	}
	var id int64
	if err := db.GetContext(ctx, &id, `INSERT INTO users(name,role,password_hash) VALUES('Existing','user','') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	sessions := customer.NewService(db, c)
	token, _, err := sessions.CreateSession(ctx, id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	session, err := sessions.Authenticate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run("link", &session); err != conflict {
		t.Fatalf("expected identity ownership conflict, got %v", err)
	}
	subject = "456"
	linked, err := run("link", &session)
	if err != nil || !linked.Linked || linked.Token != "" {
		t.Fatalf("link failed: %v", err)
	}
	result, err := run("login", nil)
	if err != nil || result.User == nil || result.User.ID != id {
		t.Fatalf("linked login failed: %v", err)
	}
}
