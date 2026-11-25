package iiko

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestFetchStopList(t *testing.T) {
	const listed = `{"terminalGroupId":"delivery","items":[{"productId":"P-1","sizeId":"size-1","balance":2},{"productId":"p-1","balance":0}]}`
	const other = `{"terminalGroupId":"other","items":[{"productId":"p-2","balance":0}]}`
	const empty = `{"terminalGroupId":"delivery","items":[]}`

	for _, tc := range []struct {
		name, group, groups string
		status              int
		want                map[string]bool
	}{
		{"configured", "DELIVERY", other + "," + listed, 200, map[string]bool{"p-1": true}},
		{"sole group", "", listed, 200, map[string]bool{"p-1": true}},
		{"empty", "delivery", empty, 200, map[string]bool{}},
		{"ambiguous", "", listed + "," + other, 200, nil},
		{"wrong group", "delivery", other, 200, nil},
		{"missing group", "", "", 200, nil},
		{"missing items", "delivery", `{"terminalGroupId":"delivery"}`, 200, nil},
		{"upstream failure", "delivery", "", 503, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v2/access_token" {
					fmt.Fprint(w, `{"token":"token"}`)
					return
				}
				if r.URL.Path != "/api/1/stop_lists" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer token" {
					t.Errorf("unexpected stop-list request: %s %s", r.Method, r.URL.Path)
				}
				var body struct {
					Organizations []string `json:"organizationIds"`
					Groups        []string `json:"terminalGroupsIds"`
					ReturnSize    bool     `json:"returnSize"`
				}
				var wantGroups []string
				if tc.group != "" {
					wantGroups = []string{tc.group}
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
					!reflect.DeepEqual(body.Organizations, []string{"org"}) ||
					!reflect.DeepEqual(body.Groups, wantGroups) || !body.ReturnSize {
					t.Errorf("unexpected request body: %+v, error: %v", body, err)
				}
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, `{"terminalGroupStopLists":[{"organizationId":"another-org","items":[%s]},{"organizationId":"org","items":[%s]}]}`, other, tc.groups)
			}))
			defer srv.Close()
			client := NewClient(srv.URL+"/api", Credentials{}, "org", "", srv.Client())
			got, err := client.FetchStopList(context.Background(), tc.group)
			if (err != nil) != (tc.want == nil) || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("FetchStopList = (%v, %v), want %v", got, err, tc.want)
			}
		})
	}
}

type stopListSource struct {
	fakeSource
	stopped map[string]bool
	stopErr error
}

func (s *stopListSource) FetchStopList(context.Context, string) (map[string]bool, error) {
	return s.stopped, s.stopErr
}

func TestSyncStopListSnapshot(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t)
	source := &stopListSource{fakeSource: fakeSource{n: oneItemMenu()}, stopped: map[string]bool{"p-1": true}}
	syncer := NewSyncer(db, source, "delivery")
	if _, err := syncer.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	var id int64
	if err := db.Get(&id, `SELECT id FROM products WHERE iiko_id='p-1'`); err != nil {
		t.Fatal(err)
	}
	s := catalog.NewStore(db)
	assertProduct := func(name string, stopped, blocked bool) {
		t.Helper()
		products, err := s.ProductsByIDs(ctx, []int64{id})
		if err != nil {
			t.Fatal(err)
		}
		if len(products) != 1 {
			t.Fatalf("expected one catalog product, got %d", len(products))
		}
		p := products[0]
		if p.Name != name || p.IsOnStopList != stopped || p.IsOrderBlocked != blocked {
			t.Fatalf("product name = %q, stop list = %t, blocked = %t", p.Name, p.IsOnStopList, p.IsOrderBlocked)
		}
	}
	item := &source.n.ItemsGroups[0].Items[0]
	assertProduct(item.Name, true, false)
	db.MustExec(`UPDATE products SET is_order_blocked=true WHERE id=$1`, id)
	originalName := item.Name
	item.Name = "New name"
	source.stopped = map[string]bool{}
	source.stopErr = errors.New("unavailable")
	if _, err := syncer.Sync(ctx); !errors.Is(err, source.stopErr) {
		t.Fatalf("expected stop-list error, got %v", err)
	}
	assertProduct(originalName, true, true)
	source.stopErr = nil
	if _, err := syncer.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	assertProduct(item.Name, false, true)

	syncedName := item.Name
	item.Name = "Unsynced menu name"
	source.stopped = map[string]bool{"p-1": true}
	if err := syncer.RefreshStopList(ctx); err != nil {
		t.Fatal(err)
	}
	assertProduct(syncedName, true, true)
	source.stopped = map[string]bool{}
	if err := syncer.RefreshStopList(ctx); err != nil {
		t.Fatal(err)
	}
	assertProduct(syncedName, false, true)
}
