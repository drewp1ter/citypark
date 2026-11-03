// Command content-seed exports and explicitly restores public content independently of demo data.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

type dataset struct {
	Columns []string        `json:"columns"`
	Rows    json.RawMessage `json:"rows"`
}
type bundle struct {
	Version int                `json:"version"`
	Tables  map[string]dataset `json:"tables"`
	Assets  map[string]string  `json:"assets"`
}

func tables() []string {
	return []string{"news", "delivery_prices", "content_blocks", "rooms", "site_settings", "delivery_settings", "working_hours", "hotel_prices", "breakfast_items"}
}
func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: content-seed export|check|restore [flags]")
	}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	dir := f.String("bundle", "seeds/production", "bundle directory")
	uploads := f.String("uploads", env("UPLOAD_DIR", "uploads"), "upload directory")
	images := f.String("images", "../citypark-modern/public/images", "website images directory")
	replace := f.Bool("replace", false, "replace selected target datasets")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if args[0] != "export" && args[0] != "check" && args[0] != "restore" {
		return fmt.Errorf("unknown subcommand")
	}
	if args[0] == "restore" && !*replace {
		return fmt.Errorf("restore requires --replace; stop application writes and back up the target first")
	}
	db, err := store.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
	if args[0] == "export" {
		return export(ctx, db, *dir, *uploads, *images)
	}
	b, err := load(*dir)
	if err != nil {
		return err
	}
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: args[0] == "check"})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = compatible(ctx, tx, b); err != nil {
		return err
	}
	if err = assets(b, *dir, *images, func(p string, data []byte) error {
		if args[0] == "check" {
			return nil
		}
		return copyFile(filepath.Join(*uploads, strings.TrimPrefix(p, "/uploads/")), data)
	}); err != nil {
		return err
	}
	if args[0] == "check" {
		fmt.Println("bundle, schema and assets validated")
		return nil
	}
	for _, name := range tables() {
		if _, err = tx.ExecContext(ctx, "DELETE FROM public."+name); err != nil {
			return err
		}
		d := b.Tables[name]
		cols := strings.Join(d.Columns, ",")
		if _, err = tx.ExecContext(ctx, "INSERT INTO public."+name+" ("+cols+") SELECT "+cols+" FROM jsonb_populate_recordset(NULL::public."+name+", $1::jsonb)", string(d.Rows)); err != nil {
			return fmt.Errorf("restore %s: %w", name, err)
		}
		if name != "news" && name != "delivery_prices" && name != "breakfast_items" {
			continue
		}
		var seq sql.NullString
		if err = tx.GetContext(ctx, &seq, "SELECT pg_get_serial_sequence($1, 'id')", "public."+name); err != nil {
			return err
		}
		if seq.Valid { // Transactional sequence restart; rollback also restores its previous state.
			var next int64
			if err = tx.GetContext(ctx, &next, "SELECT COALESCE(MAX(id),0)+1 FROM public."+name); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH %d", seq.String, next)); err != nil {
				return err
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	fmt.Println("public content restored")
	return nil
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func columns(ctx context.Context, tx *sqlx.Tx, name string) ([]string, error) {
	var c []string
	err := tx.SelectContext(ctx, &c, "SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 ORDER BY ordinal_position", name)
	return c, err
}
func compatible(ctx context.Context, tx *sqlx.Tx, b bundle) error {
	if b.Version != 1 || len(b.Tables) != len(tables()) {
		return fmt.Errorf("unsupported bundle version or datasets")
	}
	for _, n := range tables() {
		d, ok := b.Tables[n]
		if !ok {
			return fmt.Errorf("missing dataset %s", n)
		}
		c, err := columns(ctx, tx, n)
		if err != nil {
			return err
		}
		if len(c) == 0 || strings.Join(c, ",") != strings.Join(d.Columns, ",") {
			return fmt.Errorf("schema mismatch: %s", n)
		}
		var rows []map[string]json.RawMessage
		if err = json.Unmarshal(d.Rows, &rows); err != nil || rows == nil {
			return fmt.Errorf("invalid rows: %s", n)
		}
		for _, r := range rows {
			if len(r) != len(c) {
				return fmt.Errorf("incomplete row: %s", n)
			}
			for _, k := range c {
				if _, ok := r[k]; !ok {
					return fmt.Errorf("missing column %s", k)
				}
			}
		}
	}
	return nil
}
func load(dir string) (bundle, error) {
	var b bundle
	data, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(data, &b)
	return b, err
}
func references(b bundle) []string {
	re := regexp.MustCompile(`/(?:uploads|images)/[^\s"'<>\\]+`)
	found := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			for _, p := range re.FindAllString(x, -1) {
				found[p] = true
			}
		case []any:
			for _, v := range x {
				walk(v)
			}
		case map[string]any:
			for _, v := range x {
				walk(v)
			}
		}
	}
	for _, d := range b.Tables {
		var v any
		_ = json.Unmarshal(d.Rows, &v)
		walk(v)
	}
	paths := []string{}
	for p := range found {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
func safe(p string) bool {
	return (strings.HasPrefix(p, "/uploads/") || strings.HasPrefix(p, "/images/")) && filepath.ToSlash(filepath.Clean(p)) == p && !strings.Contains(p, "\\") && !strings.HasSuffix(p, "/")
}
func assets(b bundle, dir, images string, write func(string, []byte) error) error {
	paths := references(b)
	if len(paths) != len(b.Assets) {
		return fmt.Errorf("asset manifest does not match content")
	}
	for _, p := range paths {
		if !safe(p) {
			return fmt.Errorf("unsafe asset path %q", p)
		}
		src := filepath.Join(dir, "assets", p)
		if strings.HasPrefix(p, "/images/") {
			src = filepath.Join(images, strings.TrimPrefix(p, "/images/"))
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != b.Assets[p] {
			return fmt.Errorf("asset checksum mismatch: %s", p)
		}
		if strings.HasPrefix(p, "/uploads/") {
			if err = write(p, data); err != nil {
				return err
			}
		}
	}
	return nil
}
func copyFile(path string, data []byte) error {
	old, err := os.ReadFile(path)
	if err == nil {
		if string(old) != string(data) {
			return fmt.Errorf("conflicting asset %s", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	return closeErr
}
func export(ctx context.Context, db *sqlx.DB, dir, uploads, images string) error {
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return fmt.Errorf("export requires a new bundle directory")
	}
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	b := bundle{Version: 1, Tables: map[string]dataset{}, Assets: map[string]string{}}
	for _, n := range tables() {
		c, err := columns(ctx, tx, n)
		if err != nil {
			return err
		}
		key := "id"
		if n == "content_blocks" {
			key = "key"
		}
		var data []byte
		if err = tx.GetContext(ctx, &data, "SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY t."+key+"), '[]'::jsonb) FROM public."+n+" t"); err != nil {
			return err
		}
		b.Tables[n] = dataset{c, json.RawMessage(data)}
	}
	if err = compatible(ctx, tx, b); err != nil {
		return err
	}
	for _, p := range references(b) {
		if !safe(p) {
			return fmt.Errorf("unsafe asset %s", p)
		}
		root, rel := uploads, strings.TrimPrefix(p, "/uploads/")
		if strings.HasPrefix(p, "/images/") {
			root, rel = images, strings.TrimPrefix(p, "/images/")
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return err
		}
		b.Assets[p] = fmt.Sprintf("%x", sha256.Sum256(data))
		if strings.HasPrefix(p, "/uploads/") {
			if err = copyFile(filepath.Join(dir, "assets", p), data); err != nil {
				return err
			}
		}
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "snapshot.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	return tx.Commit()
}
