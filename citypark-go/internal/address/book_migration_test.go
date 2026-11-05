package address

import (
	"context"
	"os"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/migrations"
	"github.com/pressly/goose/v3"
)

func testBookMigration(t *testing.T, down bool) {
	t.Helper()
	ctx := context.Background()
	db, err := store.Open(ctx, os.Getenv("TEST_DB_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	exec := func(q string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	exec(`CREATE SCHEMA book_migration_test`)
	defer func() { exec(`SET search_path TO public`); exec(`DROP SCHEMA book_migration_test CASCADE`) }()
	exec(`SET search_path TO book_migration_test`)
	p, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpTo(ctx, 32); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO users(id,name,phone) VALUES(1,'a','1'),(2,'b','2'); INSERT INTO addresses(id,city,address) VALUES(1,'Город','  старый адрес  '); INSERT INTO user_addresses(user_id,address_id) VALUES(1,1),(2,1); INSERT INTO orders(customer_id,address_id,items_hash) VALUES(1,1,'hash')`)
	if _, err := p.UpTo(ctx, 33); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.Get(&n, `SELECT count(DISTINCT id) FROM user_addresses WHERE street IS NULL AND NOT is_default AND name='' AND created_at=updated_at`); err != nil || n != 2 {
		t.Fatal(n, err)
	}
	var text string
	if err := db.Get(&text, `SELECT a.address FROM orders o JOIN addresses a ON a.id=o.address_id`); err != nil || text != "  старый адрес  " {
		t.Fatal(text, err)
	}
	if !down {
		exec(`INSERT INTO addresses(id,city,address) VALUES(2,'Город','Новый'); INSERT INTO user_addresses(user_id,address_id) VALUES(1,2)`)
		var id int64
		if err := db.Get(&id, `SELECT id FROM user_addresses WHERE address_id=2`); err != nil || id <= 2 {
			t.Fatal(id, err)
		}
		return
	}
	for _, update := range []string{`street='Дом'`, `name='Дом'`, `is_default=true`, `apartment='1'`, `entrance='2'`} {
		exec(`UPDATE user_addresses SET ` + update + ` WHERE user_id=1`)
		if _, err := p.DownTo(ctx, 32); err == nil {
			t.Fatal("metadata discarded", update)
		}
		if err := db.Get(&n, `SELECT count(*) FROM user_addresses`); err != nil || n != 2 {
			t.Fatal(n, err)
		}
		exec(`UPDATE user_addresses SET street=NULL,name='',is_default=false,apartment='',entrance=''`)
	}
	if _, err := p.DownTo(ctx, 32); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&n, `SELECT count(*) FROM user_addresses`); err != nil || n != 2 {
		t.Fatal(n, err)
	}
}
func TestBookMigration(t *testing.T)     { testBookMigration(t, false) }
func TestBookMigrationDown(t *testing.T) { testBookMigration(t, true) }
