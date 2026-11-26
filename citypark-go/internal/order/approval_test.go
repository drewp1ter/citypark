package order

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestApplyApproval(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input approvalInput
		ok    bool
	}{
		{"valid", approvalInput{[]int64{2}, []itemComment{{1, " " + strings.Repeat("я", 255) + " "}}}, true},
		{"all excluded", approvalInput{ExcludedProductIDs: []int64{1, 2}}, false},
		{"unknown exclusion", approvalInput{ExcludedProductIDs: []int64{3}}, false},
		{"duplicate exclusion", approvalInput{ExcludedProductIDs: []int64{2, 2}}, false},
		{"excluded comment", approvalInput{[]int64{2}, []itemComment{{2, "без лука"}}}, false},
		{"unknown comment", approvalInput{ItemComments: []itemComment{{3, "без лука"}}}, false},
		{"duplicate comment", approvalInput{ItemComments: []itemComment{{1, ""}, {1, "без лука"}}}, false},
		{"long comment", approvalInput{ItemComments: []itemComment{{1, strings.Repeat("я", 256)}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := Order{Items: []Item{{ProductID: 1}, {ProductID: 2}}}
			err := applyApproval(&o, tc.input)
			if (err == nil) != tc.ok {
				t.Fatalf("applyApproval error = %v, want success %v", err, tc.ok)
			}
			if tc.ok && (!o.Items[1].ExcludedFromIiko || o.Items[0].IikoComment != strings.Repeat("я", 255)) {
				t.Fatalf("selection = %+v", o.Items)
			}
		})
	}
}

func TestApproveOrderConcurrentRequests(t *testing.T) {
	e := newAdminEnv(t)
	opts := defaultOpts()
	opts.PrepareIiko = func(o Order) ([]byte, error) { return json.Marshal(o.Items) }
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	r := chi.NewRouter()
	NewHandler(e.svc, tokens, opts).Routes(r)
	e.router = r
	path := func(id int64) string { return fmt.Sprintf("/orders/%d/approve", id) }
	if code, body := e.do(t, "POST", path(e.orderID), e.adminToken, approvalInput{}); code != http.StatusConflict {
		t.Fatalf("unpaid approval = %d, %v", code, body)
	}
	e.db.MustExec("UPDATE orders SET status='paid' WHERE id=$1", e.orderID)
	if code, body := e.do(t, "POST", path(e.orderID), e.adminToken, approvalInput{}); code != http.StatusConflict {
		t.Fatalf("historical approval = %d, %v", code, body)
	}

	id := insertOrder(t, e.db, InsertParams{
		CustomerID: testutil.InsertUser(t, e.db, "Клиент", "+79997777777", auth.RoleUser),
		Items: []InsertItem{
			{ProductID: e.productA, Amount: 1, OfferedPrice: 59900},
			{ProductID: e.productB, Amount: 1, OfferedPrice: 10000},
		},
	})
	e.db.MustExec("UPDATE products SET is_on_stop_list=true WHERE id=$1", e.productB)
	if _, err := NewStore(e.db).MarkPaid(context.Background(), id, "555"); err != nil {
		t.Fatal(err)
	}
	if code, body := e.do(t, "POST", path(id), e.userToken, approvalInput{}); code != http.StatusForbidden {
		t.Fatalf("user approval = %d, %v", code, body)
	}
	if code, body := e.do(t, "DELETE", fmt.Sprintf("/orders/%d", id), e.adminToken, nil); code != http.StatusConflict {
		t.Fatalf("delete managed paid order = %d, %v", code, body)
	}
	codes := make(chan *httptest.ResponseRecorder, 2)
	for _, comment := range []string{"Первый", "Второй"} {
		body, err := json.Marshal(approvalInput{[]int64{e.productB}, []itemComment{{e.productA, comment}}})
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			req := httptest.NewRequest("POST", path(id), strings.NewReader(string(body)))
			req.Header.Set("Authorization", "Bearer "+e.adminToken)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.router.ServeHTTP(rec, req)
			codes <- rec
		}()
	}
	var firstComment string
	for i := range 2 {
		rec := <-codes
		if rec.Code != http.StatusOK {
			t.Errorf("approval = %d, %s", rec.Code, rec.Body)
			continue
		}
		var got Order
		testutil.Decode(t, rec, &got)
		if got.IikoStatus != "queued" || got.Total != 69900 || len(got.Items) != 2 ||
			!got.Items[1].ExcludedFromIiko || !got.Items[1].IsOnStopList {
			t.Fatalf("approved order = %+v, items = %+v", got, got.Items)
		}
		if i == 0 {
			firstComment = got.Items[0].IikoComment
		} else if got.Items[0].IikoComment != firstComment {
			t.Fatal("second approval overwrote the kitchen comment")
		}
	}
}
