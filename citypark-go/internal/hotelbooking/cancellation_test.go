package hotelbooking

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCancellationCapability(t *testing.T) {
	service := new(Service)
	service.EnableCancellation("https://hotel.example", strings.Repeat("s", 32))
	config := service.cancellation
	key := "10000000-0000-4000-8000-000000000001"
	otherKey := "10000000-0000-4000-8000-000000000002"
	token := config.token(key)
	if config.link(key) != "https://hotel.example/booking/cancel#"+token || config.token(key) != token {
		t.Fatal("email retries must reuse the same URL")
	}
	r := chi.NewRouter()
	NewHandler(service, true, nil).Routes(r)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, invalid := range []string{"", otherKey + token[len(key):], token + "x"} {
			req := httptest.NewRequest(method, "/hotel/cancellation", nil)
			req.Header.Set("X-Hotel-Cancellation-Token", invalid)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusNotFound || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("%s invalid credential returned %d", method, w.Code)
			}
		}
	}
}
