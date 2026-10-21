package iiko

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// counter counts requests per path, safe for use from the server goroutine.
type counter struct {
	mu     sync.Mutex
	byPath map[string]int
}

func newCounter() *counter { return &counter{byPath: map[string]int{}} }

// inc records a request and returns how many were seen on that path so far.
func (c *counter) inc(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byPath[path]++
	return c.byPath[path]
}

func (c *counter) get(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.byPath[path]
}

func (c *counter) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, v := range c.byPath {
		n += v
	}
	return n
}

var testCreds = Credentials{APIKey: "key-1", AppID: "app-1", ClientSecret: "secret-1"}

func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, testCreds, "org-1", "menu-1", nil)
}

func TestAccessToken(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v2/access_token" {
			t.Errorf("path = %s, want /v2/access_token", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content type = %q", ct)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want empty", got)
		}
		var body struct {
			APIKey       string `json:"apiKey"`
			AppID        string `json:"appId"`
			ClientSecret string `json:"clientSecret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.APIKey != "key-1" || body.AppID != "app-1" || body.ClientSecret != "secret-1" {
			t.Errorf("body = %+v", body)
		}
		w.Write([]byte(`{"correlationId":"c1","token":"t1"}`))
	}))

	got, err := c.AccessToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "t1" {
		t.Errorf("token = %q, want t1", got)
	}
}

func TestAccessTokenUnauthorized(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	_, err := c.AccessToken(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestAccessTokenEmpty(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"token":""}`))
	}))

	got, err := c.AccessToken(context.Background())
	if err == nil {
		t.Fatalf("token = %q, want error", got)
	}
}

func TestAccessTokenUnexpectedStatus(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))

	_, err := c.AccessToken(context.Background())
	if err == nil || errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want a non-unauthorized error", err)
	}
	if want := "iiko: v2/access_token: unexpected status 502"; err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestAccessTokenBadJSON(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))

	if _, err := c.AccessToken(context.Background()); err == nil {
		t.Fatal("want error for invalid JSON")
	}
}

func TestBadBaseURL(t *testing.T) {
	// A control character makes http.NewRequestWithContext fail.
	c := NewClient("http://bad\x7fhost", testCreds, "org-1", "menu-1", nil)
	if _, err := c.AccessToken(context.Background()); err == nil {
		t.Fatal("want error for invalid base URL")
	}
}

func TestFetchSendsBearerToken(t *testing.T) {
	cnt := newCounter()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt.inc(r.URL.Path)
		switch r.URL.Path {
		case "/v2/access_token":
			w.Write([]byte(`{"token":"tok-9"}`))
		case "/menu/v3/by_id":
			if r.Header.Get("Authorization") != "Bearer tok-9" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(menuJSON))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := cnt.get("/v2/access_token"); got != 1 {
		t.Errorf("access_token requests = %d, want 1", got)
	}
	if got := cnt.get("/menu/v3/by_id"); got != 1 {
		t.Errorf("menu requests = %d, want 1", got)
	}
}

func TestFetchCachesToken(t *testing.T) {
	cnt := newCounter()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt.inc(r.URL.Path)
		if r.URL.Path == "/v2/access_token" {
			w.Write([]byte(`{"token":"t"}`))
			return
		}
		w.Write([]byte(menuJSON))
	}))

	for i := 0; i < 2; i++ {
		if _, err := c.Fetch(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if got := cnt.get("/v2/access_token"); got != 1 {
		t.Errorf("access_token requests = %d, want 1", got)
	}
	if got := cnt.get("/menu/v3/by_id"); got != 2 {
		t.Errorf("menu requests = %d, want 2", got)
	}
}

func TestFetchRefreshesToken(t *testing.T) {
	cnt := newCounter()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := cnt.inc(r.URL.Path)
		switch r.URL.Path {
		case "/v2/access_token":
			tok := "good_token"
			if n == 1 {
				tok = "bad_token"
			}
			json.NewEncoder(w).Encode(map[string]string{"token": tok})
		case "/menu/v3/by_id":
			if r.Header.Get("Authorization") != "Bearer good_token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(menuJSON))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	m, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Products) != 2 || m.Products[0].ID != "p-1" {
		t.Errorf("products = %+v", m.Products)
	}
	if got := cnt.total(); got != 4 {
		t.Errorf("requests after first Fetch = %d, want 4", got)
	}

	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := cnt.total(); got != 5 {
		t.Errorf("requests after second Fetch = %d, want 5 (token cached)", got)
	}
	if got := cnt.get("/v2/access_token"); got != 2 {
		t.Errorf("access_token requests = %d, want 2", got)
	}
}

func TestFetchGivesUp(t *testing.T) {
	cnt := newCounter()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt.inc(r.URL.Path)
		if r.URL.Path == "/v2/access_token" {
			w.Write([]byte(`{"token":"t"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))

	_, err := c.Fetch(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if got := cnt.get("/menu/v3/by_id"); got != 2 {
		t.Errorf("menu requests = %d, want 2", got)
	}
}

func TestFetchAccessTokenError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("want error when the token request fails")
	}
}

func TestFetchMenuErrorIsNotRetried(t *testing.T) {
	cnt := newCounter()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt.inc(r.URL.Path)
		if r.URL.Path == "/v2/access_token" {
			w.Write([]byte(`{"token":"t"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, err := c.Fetch(context.Background())
	if err == nil || errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want a non-unauthorized error", err)
	}
	if got := cnt.total(); got != 2 {
		t.Errorf("requests = %d, want 2 (no retry)", got)
	}
}

func TestFetchRefreshTokenError(t *testing.T) {
	cnt := newCounter()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := cnt.inc(r.URL.Path)
		if r.URL.Path == "/v2/access_token" {
			if n == 1 {
				w.Write([]byte(`{"token":"t"}`))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))

	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("want error when the token refresh fails")
	}
}

func TestFetchContextCancelled(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"token":"t"}`))
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.Fetch(ctx); err == nil {
		t.Fatal("want error for a cancelled context")
	}
}

const menuJSON = `{"id":"94764","name":"test","overrideTaxCategories":[],
 "itemsGroups":[
  {"id":"g-1","name":"Шашлыки","isHidden":true,"items":[
    {"productId":"p-1","name":"Шашлык из свинины","description":null,"menuItemId":"m-1",
     "sizePrices":[{"sizeName":"","isDefault":true,"weight":1000.0,"sku":"01117"}]}]},
  {"id":"g-2","name":"Хинкали","items":[
    {"productId":"p-2","name":"Хинкали","description":"  Говядина, лук  ","menuItemId":"m-2",
     "sizePrices":[{"sizeName":"","isDefault":true,"price":600.0,"buttonImageUrl":"http://i/p-2.jpeg","weight":1000.0,
       "nutritions":{"fats":200.0,"proteins":100.0,"carbs":300.0,"energy":400.0},
       "sku":"00047","menuItemSizeId":"s-2"}]}]}],
 "products":[{"id":"p-1","sku":"01117","measureUnit":"кг"},{"id":"p-2","sku":"00047","measureUnit":"кг"}],
 "taxCategories":[],"productCategories":[],"allergenGroups":[]}`

func TestMenu(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/menu/v3/by_id" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content type = %q", ct)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer t1" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer t1")
		}
		var body struct {
			ExternalMenuID string `json:"externalMenuId"`
			OrganizationID string `json:"organizationId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.ExternalMenuID != "menu-1" || body.OrganizationID != "org-1" {
			t.Errorf("body = %+v", body)
		}
		w.Write([]byte(menuJSON))
	}))

	m, err := c.Menu(context.Background(), "t1")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "94764" || len(m.ItemsGroups) != 2 {
		t.Fatalf("menu = %+v", m)
	}
	if !m.ItemsGroups[0].IsHidden || m.ItemsGroups[1].IsHidden {
		t.Errorf("IsHidden = %v/%v, want true/false", m.ItemsGroups[0].IsHidden, m.ItemsGroups[1].IsHidden)
	}
	if p := m.ItemsGroups[0].Items[0].SizePrices[0].Price; p != nil {
		t.Errorf("price without value = %v, want nil", *p)
	}
	if p := m.ItemsGroups[1].Items[0].SizePrices[0].Price; p == nil || *p != 600 {
		t.Errorf("price = %v, want 600", p)
	}
	if d := m.ItemsGroups[1].Items[0].Description; d != "  Говядина, лук  " {
		t.Errorf("description = %q, want the raw text with spaces", d)
	}
	if d := m.ItemsGroups[0].Items[0].Description; d != "" {
		t.Errorf("null description = %q, want empty", d)
	}
	size := m.ItemsGroups[1].Items[0].SizePrices[0]
	if size.ButtonImageURL != "http://i/p-2.jpeg" || size.MenuItemSizeID != "s-2" ||
		size.Nutritions != (Nutritions{Fats: 200, Proteins: 100, Carbs: 300, Energy: 400}) {
		t.Errorf("size = %+v", size)
	}
	if len(m.Products) != 2 || m.Products[1].MeasureUnit != "кг" {
		t.Errorf("products = %+v", m.Products)
	}
}

func TestMenuUnauthorized(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	_, err := c.Menu(context.Background(), "t1")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestMenuServerError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, err := c.Menu(context.Background(), "t1")
	if err == nil || errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want a non-unauthorized error", err)
	}
	if want := "iiko: menu/v3/by_id: unexpected status 500"; err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestFetch(t *testing.T) {
	cnt := newCounter()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := cnt.inc(r.URL.Path)
		switch r.URL.Path {
		case "/v2/access_token":
			tok := "good"
			if n == 1 {
				tok = "bad"
			}
			json.NewEncoder(w).Encode(map[string]string{"token": tok})
		case "/menu/v3/by_id":
			if r.Header.Get("Authorization") != "Bearer good" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(menuJSON))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	m, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "94764" {
		t.Errorf("menu id = %q", m.ID)
	}
	if got := cnt.get("/v2/access_token"); got != 2 {
		t.Errorf("token requests = %d, want 2", got)
	}
	if got := cnt.get("/menu/v3/by_id"); got != 2 {
		t.Errorf("menu requests = %d, want 2", got)
	}
}
