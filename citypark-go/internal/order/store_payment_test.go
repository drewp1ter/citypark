package order

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func setOrderState(t *testing.T, db *sqlx.DB, id int64, status, paymentID string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE orders SET status = $2, payment_id = $3 WHERE id = $1`, id, status, paymentID); err != nil {
		t.Fatalf("set order state: %v", err)
	}
}

func TestSetStatus(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	id := insertOrder(t, db, sd.params(true))
	s := NewStore(db)

	if err := s.SetStatus(context.Background(), id, StatusPending); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	got, err := s.ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, StatusPending)
	}
	if got.UpdatedAt.Before(got.CreatedAt) {
		t.Errorf("UpdatedAt %v is before CreatedAt %v", got.UpdatedAt, got.CreatedAt)
	}
}

func TestSetStatusUnknownOrder(t *testing.T) {
	db := testutil.NewDB(t)
	err := NewStore(db).SetStatus(context.Background(), 12345, StatusPending)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want store.ErrNotFound", err)
	}
}

func TestSetStatusInvalid(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	id := insertOrder(t, db, sd.params(true))

	err := NewStore(db).SetStatus(context.Background(), id, "bogus")
	if !store.IsCheckViolation(err) {
		t.Errorf("err = %v, want check violation", err)
	}
}

func TestMarkPaid(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		paymentID     string
		wantChanged   bool
		wantStatus    string
		wantPaymentID string
	}{
		{"new", StatusNew, "", true, StatusPaid, "555"},
		{"pending", StatusPending, "", true, StatusPaid, "555"},
		{"paid", StatusPaid, "111", false, StatusPaid, "111"},
		{"canceled", StatusCanceled, "", false, StatusCanceled, ""},
		{"confirmed", StatusConfirmed, "", false, StatusConfirmed, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			sd := seed(t, db)
			id := insertOrder(t, db, sd.params(true))
			setOrderState(t, db, id, tt.status, tt.paymentID)
			s := NewStore(db)

			changed, err := s.MarkPaid(context.Background(), id, "555")
			if err != nil {
				t.Fatalf("MarkPaid: %v", err)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}

			got, err := s.ByID(context.Background(), id)
			if err != nil {
				t.Fatalf("ByID: %v", err)
			}
			if got.Status != tt.wantStatus || got.PaymentID != tt.wantPaymentID {
				t.Errorf("status %q, payment id %q; want %q, %q",
					got.Status, got.PaymentID, tt.wantStatus, tt.wantPaymentID)
			}
		})
	}
}

func TestMarkPaidTwice(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	id := insertOrder(t, db, sd.params(true))
	s := NewStore(db)
	ctx := context.Background()

	first, err := s.MarkPaid(ctx, id, "555")
	if err != nil || !first {
		t.Fatalf("first MarkPaid = %v, %v; want true, nil", first, err)
	}
	second, err := s.MarkPaid(ctx, id, "777")
	if err != nil || second {
		t.Fatalf("second MarkPaid = %v, %v; want false, nil", second, err)
	}

	got, err := s.ByID(ctx, id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.PaymentID != "555" {
		t.Errorf("PaymentID = %q, want 555", got.PaymentID)
	}
}

func TestMarkPaidUnknownOrder(t *testing.T) {
	db := testutil.NewDB(t)
	changed, err := NewStore(db).MarkPaid(context.Background(), 12345, "555")
	if err != nil || changed {
		t.Errorf("MarkPaid = %v, %v; want false, nil", changed, err)
	}
}

func TestFindOpenDuplicate(t *testing.T) {
	const phone = "+79990000000" // the phone of the seeded user
	base := Address{City: "Москва", Address: "Тверская, 1"}
	other := Address{City: "Москва", Address: "Арбат, 5"}

	tests := []struct {
		name        string
		withAddress bool
		status      string
		payment     string
		comment     string
		hashChange  bool
		phone       string
		addr        *Address
		want        bool
	}{
		{name: "same", withAddress: true, status: StatusNew, addr: &base, want: true},
		{name: "pending", withAddress: true, status: StatusPending, addr: &base, want: true},
		{name: "paid", withAddress: true, status: StatusPaid, addr: &base, want: false},
		{name: "cash", withAddress: true, status: StatusNew, payment: PaymentCash, addr: &base, want: false},
		{name: "another comment", withAddress: true, status: StatusNew, comment: "другой", addr: &base, want: false},
		{name: "another hash", withAddress: true, status: StatusNew, hashChange: true, addr: &base, want: false},
		{name: "another phone", withAddress: true, status: StatusNew, phone: "+79991112233", addr: &base, want: false},
		{name: "another address", withAddress: true, status: StatusNew, addr: &other, want: false},
		{name: "nil address, order has one", withAddress: true, status: StatusNew, addr: nil, want: false},
		{name: "address given, order has none", withAddress: false, status: StatusNew, addr: &base, want: false},
		{name: "nil address, order has none", withAddress: false, status: StatusNew, addr: nil, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			sd := seed(t, db)
			p := sd.params(tt.withAddress)
			p.PaymentType = PaymentOnline
			if tt.payment != "" {
				p.PaymentType = tt.payment
			}
			id := insertOrder(t, db, p)
			setOrderState(t, db, id, tt.status, "")

			comment := p.Comment
			if tt.comment != "" {
				comment = tt.comment
			}
			hash := p.ItemsHash
			if tt.hashChange {
				hash = ItemsHash([]ItemInput{{ProductID: sd.productA, Amount: 9}})
			}
			ph := phone
			if tt.phone != "" {
				ph = tt.phone
			}

			gotID, found, err := NewStore(db).FindOpenDuplicate(context.Background(), ph, tt.addr, comment, hash)
			if err != nil {
				t.Fatalf("FindOpenDuplicate: %v", err)
			}
			if found != tt.want {
				t.Fatalf("found = %v, want %v", found, tt.want)
			}
			if found && gotID != id {
				t.Errorf("id = %d, want %d", gotID, id)
			}
			if !found && gotID != 0 {
				t.Errorf("id = %d, want 0 when not found", gotID)
			}
		})
	}
}

func TestFindOpenDuplicateReturnsHighestID(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	p := sd.params(true)
	p.PaymentType = PaymentOnline
	insertOrder(t, db, p)
	second := insertOrder(t, db, p)

	addr := &Address{City: "Москва", Address: "Тверская, 1"}
	id, found, err := NewStore(db).FindOpenDuplicate(context.Background(), "+79990000000", addr, p.Comment, p.ItemsHash)
	if err != nil {
		t.Fatalf("FindOpenDuplicate: %v", err)
	}
	if !found || id != second {
		t.Errorf("id, found = %d, %v; want %d, true", id, found, second)
	}
}
