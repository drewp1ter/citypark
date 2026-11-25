package hotelbooking

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSMSAero(t *testing.T) {
	for _, tc := range []struct {
		name      string
		response  string
		wantError bool
	}{
		{"accepted", `{"success":true,"data":{"id":42}}`, false},
		{"rejected", `{"success":false,"data":null}`, true},
		{"timeout", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				login, key, _ := r.BasicAuth()
				var payload map[string]string
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil ||
					r.Method != http.MethodPost || login != "login" || key != "key" ||
					payload["number"] != "79990000000" || payload["sign"] != "CityPark" ||
					payload["channel"] != "DIRECT" || payload["text"] != "Бронь подтверждена" {
					t.Error("unexpected SMS request")
				}
				if tc.response == "" {
					<-r.Context().Done()
					return
				}
				_, _ = io.WriteString(w, tc.response)
			}))
			defer server.Close()
			client := NewSMSAero("login", "key")
			client.endpoint = server.URL
			client.client.Timeout = 250 * time.Millisecond
			err := client.Send(context.Background(), "+79990000000", "Бронь подтверждена")
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
}
