package customer_test

import (
	"context"
	"errors"
	"github.com/drewp1ter/citypark-go/internal/address"
	"reflect"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/jmoiron/sqlx"
)

func mergeService(db *sqlx.DB) *customer.Service {
	return customer.NewService(db, clock.Fixed(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)))
}

func mustExec(t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func TestMergeAccounts(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	source := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	destination := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	guest := testutil.InsertUser(t, db, "Guest", "+79990000003", "user")
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','101'),($2,'google','g1')`, source, destination)
	a := testutil.InsertAddress(t, db, source, "City", "One")
	testutil.InsertAddress(t, db, source, "City", "Two")
	mustExec(t, db, `INSERT INTO user_addresses(user_id,address_id) VALUES($1,$2)`, destination, a)
	mustExec(t, db, `INSERT INTO orders(customer_id,account_id,contact_name,contact_phone,items_hash) VALUES($1,$1,'Mini','snapshot-a','a'),($2,$2,'Website','snapshot-b','b'),($3,NULL,'Guest','snapshot-c','c')`, source, destination, guest)
	mustExec(t, db, `INSERT INTO customer_email_tokens(token_hash,user_id,purpose,email,expires_at) VALUES($1,$2,'verify','mini@example.com',now()+interval '1 day')`, make([]byte, 32), source)
	mustExec(t, db, `INSERT INTO customer_oauth_attempts(state_hash,nonce_hash,flow_hash,pkce_verifier,intent,user_id,session_hash,expires_at) VALUES($1,$2,$3,'secret','link',$4,$5,now()+interval '1 day')`, bytes32(1), bytes32(2), bytes32(3), source, bytes32(4))
	s := mergeService(db)
	sourceToken, _, err := s.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	destinationToken, _, err := s.CreateSession(ctx, destination, "google", "web")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MergeAccounts(ctx, source, destination); err != nil {
		t.Fatal(err)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM customer_identities WHERE user_id=$1`, destination) != 2 ||
		testutil.Count(t, db, `SELECT count(*) FROM orders WHERE account_id=$1`, destination) != 2 ||
		testutil.Count(t, db, `SELECT count(*) FROM orders WHERE account_id IS NULL AND customer_id=$1`, guest) != 1 ||
		testutil.Count(t, db, `SELECT count(*) FROM user_addresses WHERE user_id=$1`, destination) != 2 ||
		testutil.Count(t, db, `SELECT count(*) FROM user_addresses WHERE user_id=$1`, source) != 0 {
		t.Fatal("ownership or addresses were not transferred correctly")
	}
	if testutil.Count(t, db, `SELECT count(*) FROM orders WHERE customer_id=$1 AND contact_phone='snapshot-a' AND account_id=$2`, source, destination) != 1 {
		t.Fatal("checkout snapshot changed")
	}
	if testutil.Count(t, db, `SELECT count(*) FROM customer_email_tokens WHERE user_id=$1 AND consumed_at IS NOT NULL`, source) != 1 ||
		testutil.Count(t, db, `SELECT count(*) FROM customer_oauth_attempts WHERE user_id=$1 AND consumed_at IS NOT NULL AND pkce_verifier=''`, source) != 1 {
		t.Fatal("source proofs remain usable")
	}
	if _, err := s.Authenticate(ctx, sourceToken); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("source session: %v", err)
	}
	if _, err := s.Authenticate(ctx, destinationToken); err != nil {
		t.Fatalf("destination session: %v", err)
	}
	if _, _, err := s.CreateSession(ctx, source, "telegram", "mini"); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("merged source session: %v", err)
	}
}

func bytes32(v byte) []byte { b := make([]byte, 32); b[0] = v; return b }

func TestMergeCredentialTransfer(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	source := testutil.InsertUser(t, db, "Source", "+79990000001", "user")
	destination := testutil.InsertUser(t, db, "Destination", "+79990000002", "user")
	hash, err := auth.HashPassword("password-123")
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `UPDATE users SET email='source@example.com',password_hash=$2,is_email_confirmed=true WHERE id=$1`, source, hash)
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'email','source@example.com'),($2,'telegram','202')`, source, destination)
	s := mergeService(db)
	s.ConfigureEmail(&fakeMailer{}, "https://site.example", nil, nil)
	resetToken, err := s.IssueEmailToken(ctx, source, "reset", "source@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MergeAccounts(ctx, source, destination); err != nil {
		t.Fatal(err)
	}
	u, err := user.NewStore(db).ByID(ctx, destination)
	if err != nil || u.Name != "Destination" || u.Phone != "+79990000002" || u.Email == nil || *u.Email != "source@example.com" || !u.IsEmailConfirmed || !auth.CheckPassword(u.PasswordHash, "password-123") {
		t.Fatalf("destination credentials/profile: %+v %v", u, err)
	}
	u, err = user.NewStore(db).ByID(ctx, source)
	if err != nil || u.Email != nil || u.PasswordHash != "" || u.IsEmailConfirmed != false || u.Name != "Source" {
		t.Fatalf("source: %+v %v", u, err)
	}
	if err := user.NewStore(db).SetPasswordHash(ctx, source, hash); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("source credential revived: %v", err)
	}
	if err := customer.NewStore(db).LockAccount(ctx, source); !errors.Is(err, customer.ErrMergedAccount) {
		t.Fatalf("source lock: %v", err)
	}
	if err := s.ResetPassword(ctx, resetToken, "new-password-123"); !errors.Is(err, customer.ErrInvalidEmailToken) {
		t.Fatalf("source reset proof: %v", err)
	}
}

func TestMergeConflicts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, *sqlx.DB, int64, int64)
	}{
		{"two passwords", func(t *testing.T, db *sqlx.DB, a, b int64) {
			mustExec(t, db, `UPDATE users SET password_hash='one' WHERE id=ANY($1)`, []int64{a, b})
		}},
		{"provider", func(t *testing.T, db *sqlx.DB, a, b int64) {
			mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','a'),($2,'telegram','b')`, a, b)
		}},
		{"email reservation", func(t *testing.T, db *sqlx.DB, a, b int64) {
			mustExec(t, db, `UPDATE users SET email='a@example.com' WHERE id=$1`, a)
			mustExec(t, db, `UPDATE users SET email='b@example.com' WHERE id=$1`, b)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			a := testutil.InsertUser(t, db, "A", "+79990000001", "user")
			b := testutil.InsertUser(t, db, "B", "+79990000002", "user")
			tc.setup(t, db, a, b)
			if err := mergeService(db).MergeAccounts(context.Background(), a, b); !errors.Is(err, customer.ErrMergeConflict) {
				t.Fatalf("merge: %v", err)
			}
			if testutil.Count(t, db, `SELECT count(*) FROM users WHERE merged_into_user_id IS NOT NULL`) != 0 {
				t.Fatal("partial merge")
			}
		})
	}
	db := testutil.NewDB(t)
	a := testutil.InsertUser(t, db, "A", "+79990000001", "user")
	b := testutil.InsertUser(t, db, "B", "+79990000002", "user")
	admin := testutil.InsertUser(t, db, "Admin", "+79990000003", "admin")
	s := mergeService(db)
	for _, pair := range [][2]int64{{a, a}, {0, b}, {a, 99999}, {admin, b}, {a, admin}} {
		if err := s.MergeAccounts(context.Background(), pair[0], pair[1]); !errors.Is(err, customer.ErrMergeForbidden) {
			t.Fatalf("%v: %v", pair, err)
		}
	}
	if err := s.MergeAccounts(context.Background(), a, b); err != nil {
		t.Fatal(err)
	}
	if err := s.MergeAccounts(context.Background(), a, admin); !errors.Is(err, customer.ErrMergeForbidden) {
		t.Fatalf("merged source: %v", err)
	}
	if err := s.MergeAccounts(context.Background(), admin, a); !errors.Is(err, customer.ErrMergeForbidden) {
		t.Fatalf("merged destination: %v", err)
	}
}

func TestMergeConcurrencyAndCallerRollback(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	a := testutil.InsertUser(t, db, "A", "+79990000001", "user")
	b := testutil.InsertUser(t, db, "B", "+79990000002", "user")
	s := mergeService(db)
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MergeAccountsInTx(ctx, tx, a, b); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM users WHERE merged_into_user_id IS NOT NULL`) != 0 {
		t.Fatal("caller rollback failed")
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() { <-start; results <- s.MergeAccounts(ctx, a, b) }()
	go func() { <-start; results <- s.MergeAccounts(ctx, b, a) }()
	close(start)
	var success, forbidden int
	for range 2 {
		select {
		case err := <-results:
			if err == nil {
				success++
			} else if errors.Is(err, customer.ErrMergeForbidden) {
				forbidden++
			} else {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("merge deadlocked")
		}
	}
	if success != 1 || forbidden != 1 {
		t.Fatalf("opposite merges: success=%d forbidden=%d", success, forbidden)
	}
}

func TestMergeBlocksConcurrentIssuance(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	source := testutil.InsertUser(t, db, "Source", "+79990000001", "user")
	destination := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	s := mergeService(db)
	s.ConfigureEmail(&fakeMailer{}, "https://site.example", nil, nil)
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MergeAccountsInTx(ctx, tx, source, destination); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	sessionResult := make(chan error, 1)
	proofResult := make(chan error, 1)
	go func() { close(start); _, _, e := s.CreateSession(ctx, source, "telegram", "mini"); sessionResult <- e }()
	<-start
	go func() { _, e := s.IssueEmailToken(ctx, source, "verify", "source@example.com"); proofResult <- e }()
	// Both mutations must wait for the account row, then reject the merged source.
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-sessionResult:
		if !errors.Is(err, customer.ErrInvalidSession) {
			t.Fatalf("session after merge: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("session deadlocked")
	}
	select {
	case err := <-proofResult:
		if !errors.Is(err, customer.ErrInvalidEmailToken) {
			t.Fatalf("proof after merge: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("proof deadlocked")
	}
	if testutil.Count(t, db, `SELECT count(*) FROM customer_sessions WHERE user_id=$1 AND revoked_at IS NULL`, source) != 0 ||
		testutil.Count(t, db, `SELECT count(*) FROM customer_email_tokens WHERE user_id=$1 AND consumed_at IS NULL`, source) != 0 {
		t.Fatal("source credentials revived")
	}
}

func TestMergeAccountsAddressMetadata(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	source := testutil.InsertUser(t, db, "Mini", "1", "user")
	dest := testutil.InsertUser(t, db, "Web", "2", "user")
	books := address.NewService(db)
	original, _, err := books.CreateEntry(ctx, source, address.Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Home", City: "City", Street: "Street 1", Apartment: "2", Entrance: "3"})
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `INSERT INTO orders(customer_id,account_id,items_hash,delivery_address) VALUES($1,$1,'hash','historical snapshot')`, source)
	session, _, err := mergeService(db).CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	if err := mergeService(db).MergeAccounts(ctx, source, dest); err != nil {
		t.Fatal(err)
	}
	got, err := address.NewStore(db).EntryByID(ctx, dest, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	original.UpdatedAt = got.UpdatedAt
	if got != original {
		t.Fatalf("metadata changed: %+v", got)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM orders WHERE account_id=$1 AND customer_id=$2 AND delivery_address='historical snapshot'`, dest, source) != 1 {
		t.Fatal("order snapshot changed")
	}
	if _, err := mergeService(db).Authenticate(ctx, session); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatal(err)
	}
}

func TestMergeAccountsAddressRollback(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	source := testutil.InsertUser(t, db, "Mini", "1", "user")
	dest := testutil.InsertUser(t, db, "Web", "2", "user")
	testutil.InsertAddress(t, db, source, "City", "One")
	testutil.InsertAddress(t, db, source, "City", "Two")
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','123')`, source)
	mustExec(t, db, `INSERT INTO orders(customer_id,account_id,items_hash,delivery_address) VALUES($1,$1,'hash','snapshot')`, source)
	before, err := address.NewStore(db).ListEntries(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `CREATE FUNCTION fail_customer_address_merge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.user_id<>OLD.user_id AND NEW.address_id=(SELECT max(address_id) FROM user_addresses) THEN RAISE EXCEPTION 'forced failure'; END IF; RETURN NEW; END $$`)
	mustExec(t, db, `CREATE TRIGGER fail_customer_address_merge BEFORE UPDATE ON user_addresses FOR EACH ROW EXECUTE FUNCTION fail_customer_address_merge()`)
	defer mustExec(t, db, `DROP FUNCTION fail_customer_address_merge() CASCADE`)
	svc := mergeService(db)
	miniToken, _, err := svc.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	webToken, _, err := svc.CreateSession(ctx, dest, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	mini, err := svc.Authenticate(ctx, miniToken)
	if err != nil {
		t.Fatal(err)
	}
	web, err := svc.Authenticate(ctx, webToken)
	if err != nil {
		t.Fatal(err)
	}
	pairing, err := svc.CreatePairing(ctx, mini)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RedeemPairing(ctx, pairing.Code, web); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmPairing(ctx, pairing.ID, mini); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmPairing(ctx, pairing.ID, web); err == nil {
		t.Fatal("merge succeeded")
	}
	status, err := svc.GetPairing(ctx, pairing.ID, web)
	if err != nil || status.Status != "pending" || status.DestinationConfirmed {
		t.Fatalf("pairing consumed: %+v %v", status, err)
	}
	after, err := address.NewStore(db).ListEntries(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) || testutil.Count(t, db, `SELECT count(*) FROM users WHERE merged_into_user_id IS NOT NULL`) != 0 || testutil.Count(t, db, `SELECT count(*) FROM customer_identities WHERE user_id=$1`, source) != 1 || testutil.Count(t, db, `SELECT count(*) FROM orders WHERE account_id=$1 AND delivery_address='snapshot'`, source) != 1 {
		t.Fatal("partial merge")
	}
}
