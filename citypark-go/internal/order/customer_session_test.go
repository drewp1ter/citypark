package order

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestCustomerOrderAndAddressAccess(t *testing.T) {
	e := newReadEnv(t)
	ctx := context.Background()
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	sessions := customer.NewService(e.db, clock.Fixed(friday))
	token, _, err := sessions.CreateSession(ctx, e.userID, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	adapter := &customer.Adapter{Sessions: sessions, Tokens: tokens, Enabled: true}
	r := chi.NewRouter()
	NewHandler(e.svc, tokens, defaultOpts(), adapter.Required(true), adapter.Optional).Routes(r)
	for _, tc := range []struct {
		method, path, credential string
		want                     int
	}{
		{"GET", "/orders/my", token, 200},
		{"GET", fmt.Sprintf("/orders/%d", e.userOrders[0]), token, 200},
		{"GET", fmt.Sprintf("/orders/%d", e.otherOrder), token, 404},
		{"GET", "/orders/my", e.userToken, 401},
		{"GET", "/orders/my", e.adminToken, 200},
		{"POST", "/orders", "invalid", 401},
	} {
		rec := testutil.DoJSON(t, r, tc.method, tc.path, tc.credential, nil)
		if rec.Code != tc.want {
			t.Errorf("%s %s = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}
