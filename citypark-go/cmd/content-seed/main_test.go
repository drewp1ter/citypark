package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestProductionBundle(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t)
	t.Setenv("DATABASE_URL", os.Getenv("TEST_DB_DSN"))
	dir := "../../seeds/production"
	images := "../../../citypark-modern/public/images"
	uploads := t.TempDir()
	args := func(action string) []string {
		return []string{action, "--bundle", dir, "--images", images, "--uploads", uploads}
	}
	if err := run(ctx, args("check")); err != nil {
		t.Fatal(err)
	}
	if err := run(ctx, args("restore")); err == nil {
		t.Fatal("restore accepted without --replace")
	}
	if _, err := db.Exec(`INSERT INTO addresses(city,address) VALUES ('test','preserve')`); err != nil {
		t.Fatal(err)
	}
	restore := append(args("restore"), "--replace")
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(`INSERT INTO news(slug,date,title) VALUES ('target-only', '2026-01-01', 'remove') ON CONFLICT (slug) DO NOTHING`); err != nil {
			t.Fatal(err)
		}
		if err := run(ctx, restore); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := db.Get(&count, `SELECT count(*) FROM news`); err != nil || count != 76 {
			t.Fatalf("news %d: %v", count, err)
		}
		if err := db.Get(&count, `SELECT count(*) FROM addresses WHERE address='preserve'`); err != nil || count != 1 {
			t.Fatalf("excluded addresses %d: %v", count, err)
		}
	}
	out := filepath.Join(t.TempDir(), "bundle")
	if err := export(ctx, db, out, uploads, images); err != nil {
		t.Fatal(err)
	}
	original, err := load(dir)
	if err != nil {
		t.Fatal(err)
	}
	roundtrip, err := load(out)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(original)
	b, _ := json.Marshal(roundtrip)
	if string(a) != string(b) {
		t.Fatal("round trip changed snapshot")
	}
	var id int64
	if err := db.Get(&id, `INSERT INTO news(slug,date,title) VALUES ('sequence-check','2026-01-01','test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	// An invalid final dataset must roll back earlier table replacements.
	d := roundtrip.Tables["breakfast_items"]
	d.Rows = json.RawMessage(`[{"id":1,"sort":1,"name":"bad","short_name":"","price_kopecks":-1,"desktop_only":false}]`)
	roundtrip.Tables["breakfast_items"] = d
	raw, _ := json.Marshal(roundtrip)
	if err := os.WriteFile(filepath.Join(out, "snapshot.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err := run(ctx, []string{"restore", "--bundle", out, "--images", images, "--uploads", uploads, "--replace"}); err == nil {
		t.Fatal("invalid restore succeeded")
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM news WHERE slug='sequence-check'`); err != nil || count != 1 {
		t.Fatal("failed restore did not roll back")
	}
}
func TestAssetValidation(t *testing.T) {
	for _, p := range []string{"/uploads/../secret", "/uploads/a\\b", "/images//a", "/other/a"} {
		if safe(p) {
			t.Errorf("accepted %s", p)
		}
	}
	b, err := load("../../seeds/production")
	if err != nil {
		t.Fatal(err)
	}
	if err = assets(b, t.TempDir(), "../../../citypark-modern/public/images", func(string, []byte) error { return nil }); err == nil {
		t.Fatal("missing assets accepted")
	}
	for p := range b.Assets {
		b.Assets[p] = "bad"
		break
	}
	if err = assets(b, "../../seeds/production", "../../../citypark-modern/public/images", func(string, []byte) error { return nil }); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
	p := filepath.Join(t.TempDir(), "asset")
	if err = copyFile(p, []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err = copyFile(p, []byte("b")); err == nil {
		t.Fatal("conflict overwritten")
	}
}
