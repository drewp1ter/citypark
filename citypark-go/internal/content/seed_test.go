package content

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/upload"
)

// seededBlocks is the number of entries in citypark-modern/src/entities/content/model/blocks.data.ts (46 on the home
// page and 55 on the hotel page), written under citypark-modern task 16 in docs/tasks/PROGRESS.md.
const (
	seededBlocks = 101
	seededPages  = 2
)

// seededDB returns the migrated database. NewDB does not truncate content_blocks, so the migration's rows are there.
func seededDB(t *testing.T) (*sqlx.DB, *upload.Storage) {
	t.Helper()
	return testutil.NewDB(t), upload.NewStorage(t.TempDir(), testBase)
}

func TestSeedBlocks(t *testing.T) {
	db, uploads := seededDB(t)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)))
	r := chi.NewRouter()
	NewHandler(db, uploads, tokens).Routes(r)

	rec := testutil.DoJSON(t, r, http.MethodGet, "/content/blocks", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var blocks []map[string]any
	testutil.Decode(t, rec, &blocks)
	if len(blocks) != seededBlocks {
		t.Fatalf("got %d blocks, want %d", len(blocks), seededBlocks)
	}
	byKey := map[string]map[string]any{}
	for _, b := range blocks {
		byKey[b["key"].(string)] = b
	}

	text := byKey["home.hero.lead"]
	if text["type"] != "text" || text["page"] != "home" || text["section"] != "Первый экран" ||
		text["label"] != "Подзаголовок" || text["maxLength"] != float64(200) ||
		text["value"] != "Всё в одном месте — у Городского парка. Европейская и японская кухня, 12 номеров «Люкс»." {
		t.Errorf("home.hero.lead = %v", text)
	}

	lines := byKey["hotel.quick-facts"]
	wantLines := []any{
		"Заезд 24/7, выезд до 12:00",
		"Наличные или карта, без предоплаты",
		"Без животных · не курить (штраф {{smokingFine}})",
	}
	if lines["type"] != "lines" || lines["page"] != "hotel" || !reflect.DeepEqual(lines["value"], wantLines) {
		t.Errorf("hotel.quick-facts = %v", lines)
	}

	image := byKey["hotel.extras.photo"]
	wantImage := map[string]any{"url": uploads.Public("/images/hotel-park.jpg"), "alt": "Гостиница среди деревьев", "position": ""}
	if image["type"] != "image" || !reflect.DeepEqual(image["image"], wantImage) {
		t.Errorf("hotel.extras.photo = %v, want image %v", image, wantImage)
	}

	rich := byKey["hotel.rules.2.text"]
	wantRich := "<p>До 6&nbsp;ч — {{lateCheckoutPerHour}}/час · 6–12&nbsp;ч — половина суток · 12–24&nbsp;ч — сутки.</p>"
	if rich["type"] != "richtext" || rich["value"] != wantRich {
		t.Errorf("hotel.rules.2.text = %v", rich)
	}

	empty := byKey["hotel.guest-info.2.text"]
	if empty["type"] != "richtext" || empty["value"] != "" {
		t.Errorf("hotel.guest-info.2.text = %v, want an empty richtext", empty)
	}
}

func TestSeedBlocksValid(t *testing.T) {
	db, uploads := seededDB(t)
	blocks, err := NewStore(db).List(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != seededBlocks {
		t.Fatalf("got %d blocks, want %d", len(blocks), seededBlocks)
	}
	for _, b := range blocks {
		var in valueInput
		switch b.Type {
		case TypeText, TypeRichtext:
			raw, _ := json.Marshal(b.TextValue)
			in.Value = raw
		case TypeLines:
			lines := []string{}
			if b.TextValue != "" {
				lines = strings.Split(b.TextValue, "\n")
			}
			raw, _ := json.Marshal(lines)
			in.Value = raw
		case TypeImage:
			in.Image = &imageInput{URL: b.ImageURL, Alt: b.ImageAlt, Position: b.ImagePosition}
			in.uploads = uploads
		}
		got, err := validateValue(b, in)
		if err != nil {
			t.Errorf("%s: its own value is rejected: %v", b.Key, err)
			continue
		}
		// The sanitizer turns &nbsp; into a literal no-break space, so a richtext value is only checked for acceptance.
		if (b.Type == TypeText || b.Type == TypeLines) && got != b.TextValue {
			t.Errorf("%s: validated value %q differs from the stored %q", b.Key, got, b.TextValue)
		}
	}
}

func TestSeedKeysUnique(t *testing.T) {
	db, _ := seededDB(t)
	var pages, rows, keys int
	err := db.QueryRow(`SELECT count(DISTINCT page), count(*), count(DISTINCT key) FROM content_blocks`).
		Scan(&pages, &rows, &keys)
	if err != nil {
		t.Fatal(err)
	}
	if pages != seededPages {
		t.Errorf("pages = %d, want %d", pages, seededPages)
	}
	if rows != seededBlocks || keys != rows {
		t.Errorf("rows = %d, distinct keys = %d, want %d of both", rows, keys, seededBlocks)
	}
	var badSort int
	err = db.Get(&badSort, `SELECT count(*) FROM (SELECT page, sort FROM content_blocks GROUP BY page, sort
		HAVING count(*) > 1) d`)
	if err != nil {
		t.Fatal(err)
	}
	if badSort != 0 {
		t.Errorf("%d (page, sort) pairs are used twice", badSort)
	}
}
