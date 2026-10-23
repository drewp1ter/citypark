package content

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/richtext"
)

func guestInfoTexts(t *testing.T) map[string]Block {
	t.Helper()
	db, _ := seededDB(t)
	blocks, err := NewStore(db).List(context.Background(), "hotel")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	out := map[string]Block{}
	for _, b := range blocks {
		out[b.Key] = b
	}
	return out
}

func TestOldSiteGuestInfoFilled(t *testing.T) {
	blocks := guestInfoTexts(t)
	for i := 2; i <= 6; i++ {
		key := fmt.Sprintf("hotel.guest-info.%d.text", i)
		b, ok := blocks[key]
		if !ok {
			t.Fatalf("%s: block not found", key)
		}
		if b.TextValue == "" {
			t.Errorf("%s: text_value is empty", key)
			continue
		}
		raw, err := json.Marshal(b.TextValue)
		if err != nil {
			t.Fatal(err)
		}
		got, err := validateValue(b, valueInput{Value: raw})
		if err != nil {
			t.Errorf("%s: validateValue: %v", key, err)
			continue
		}
		if got != b.TextValue {
			t.Errorf("%s: sanitizing changes the stored value\nstored: %s\nclean:  %s", key, b.TextValue, got)
		}
		if b.MaxLength != 6000 {
			t.Errorf("%s: max_length = %d, want 6000", key, b.MaxLength)
		}
		if strings.Contains(b.TextValue, "&nbsp;") || strings.Contains(b.TextValue, "<table") {
			t.Errorf("%s: contains markup that the old site had", key)
		}
	}
}

func TestOldSiteGuestInfoContent(t *testing.T) {
	blocks := guestInfoTexts(t)
	tests := []struct {
		key  string
		want []string
	}{
		{"hotel.guest-info.2.text", []string{"«Люкс»* — 8 000 ₽", "№ 23/АА-165-2021/0091-2022"}},
		{"hotel.guest-info.3.text", []string{"{{breakfastDelivery}}", "Сырники", "Лобби-бар", "Прачечная"}},
		{"hotel.guest-info.4.text", []string{"Исполнитель вправе отменить бронь"}},
		{"hotel.guest-info.5.text", []string{"Правила проживания", "Пожарная безопасность", "телефону «01»"}},
		{"hotel.guest-info.6.text", []string{"Халат — 2 000 ₽", "Занавески — 5 000 ₽"}},
	}
	for _, tc := range tests {
		for _, want := range tc.want {
			if !strings.Contains(blocks[tc.key].TextValue, want) {
				t.Errorf("%s: does not contain %q", tc.key, want)
			}
		}
	}
}

func TestOldSiteRoomDescriptions(t *testing.T) {
	db, _ := seededDB(t)
	var rooms []struct {
		ID          string `db:"id"`
		Description string `db:"description"`
	}
	if err := db.SelectContext(context.Background(), &rooms, `SELECT id, description FROM rooms ORDER BY sort`); err != nil {
		t.Fatalf("select rooms: %v", err)
	}
	if len(rooms) != 5 {
		t.Fatalf("got %d rooms, want 5", len(rooms))
	}
	for _, r := range rooms {
		if r.Description == "" {
			t.Errorf("%s: description is empty", r.ID)
		}
		if got := richtext.Sanitize(richtext.Short, r.Description); got != r.Description {
			t.Errorf("%s: sanitizing changes the description\nstored: %s\nclean:  %s", r.ID, r.Description, got)
		}
	}
}
