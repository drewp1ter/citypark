package address

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestBookFields(t *testing.T) {
	valid := Fields{City: "Город", Street: "Дом"}
	if got := FieldErrors(valid); len(got) != 0 {
		t.Fatal(got)
	}
	for _, tc := range []struct {
		field   string
		max     int
		message string
	}{{"name", 40, "Не больше 40 символов"}, {"city", 80, "Не больше 80 символов"}, {"street", 120, "Слишком длинный адрес"}, {"apartment", 20, "Не больше 20 символов"}, {"entrance", 20, "Не больше 20 символов"}} {
		for _, extra := range []int{0, 1} {
			f := valid
			value := strings.Repeat("я", tc.max+extra)
			switch tc.field {
			case "name":
				f.Name = value
			case "city":
				f.City = value
			case "street":
				f.Street = value
			case "apartment":
				f.Apartment = value
			case "entrance":
				f.Entrance = value
			}
			want := map[string]string{}
			if extra == 1 {
				want[tc.field] = tc.message
			}
			if got := FieldErrors(f); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: %v", tc.field, got)
			}
		}
	}
	if got := FieldErrors(Fields{}); !reflect.DeepEqual(got, map[string]string{"city": "Выберите населённый пункт", "street": "Укажите улицу и дом"}) {
		t.Fatal(got)
	}
	in := Fields{Name: "\u00a0Дом  мой\t", City: " Город ", Street: " Дом ", Apartment: " 1 ", Entrance: " 2 "}
	want := Fields{"Дом  мой", "Город", "Дом", "1", "2"}
	if got := CleanFields(in); got != want {
		t.Fatal(got)
	}
}
func TestFormatAddress(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		f := Fields{}
		parts := []string{}
		if mask&1 != 0 {
			f.Street = "Дом"
			parts = append(parts, "Дом")
		}
		if mask&2 != 0 {
			f.Apartment = "1"
			parts = append(parts, "кв. 1")
		}
		if mask&4 != 0 {
			f.Entrance = "2"
			parts = append(parts, "подъезд 2")
		}
		if got := FormatAddress(f); got != strings.Join(parts, ", ") {
			t.Fatal(got)
		}
	}
}
func TestDuplicateKey(t *testing.T) {
	key := DuplicateKey("Й ГОРОД", "Дом, кв. 1, подъезд 2")
	if key != DuplicateKey(" и\u0306\u00a0город ", " ДОМ,  кв. 1, подъезд 2 ") {
		t.Fatal(key)
	}
	for _, s := range []string{"Дом, кв. 2, подъезд 2", "Дом, кв. 1, подъезд 3", "Дом кв. 1, подъезд 2"} {
		if DuplicateKey("Й ГОРОД", s) == key {
			t.Fatal(s)
		}
	}
	var tuple [2]string
	if err := json.Unmarshal([]byte(key), &tuple); err != nil || tuple[0] != "й город" {
		t.Fatal(tuple, err)
	}
}
func TestListEntries(t *testing.T) { testEntries(t) }

func TestEntryByID(t *testing.T) { testEntries(t) }

func testEntries(t *testing.T) {
	t.Helper()
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := NewStore(db)
	if _, err := db.Exec(`INSERT INTO users(id,name,phone) VALUES(1,'a','1'),(2,'b','2'); INSERT INTO addresses(id,city,address) VALUES(1,'Город','  старый текст  '),(2,'Город','Дом, кв. 1'); INSERT INTO user_addresses(user_id,address_id) VALUES(1,1),(2,1); INSERT INTO user_addresses(user_id,address_id,street,apartment,is_default) VALUES(1,2,'Дом','1',true)`); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListEntries(ctx, 1)
	if err != nil || len(list) != 2 {
		t.Fatal(list, err)
	}
	if list[0].ID >= list[1].ID || !list[0].NeedsReview || list[0].Street != "" || list[0].Address != "  старый текст  " || list[1].NeedsReview || !list[1].IsDefault || list[1].Apartment != "1" || list[0].CreatedAt.IsZero() {
		t.Fatal(list)
	}
	data, _ := json.Marshal(list)
	if strings.Contains(string(data), "addressId") || !strings.Contains(string(data), `"needsReview":true`) {
		t.Fatal(string(data))
	}
	empty, err := s.ListEntries(ctx, 99)
	data, _ = json.Marshal(empty)
	if err != nil || string(data) != "[]" {
		t.Fatal(string(data), err)
	}
	own, err := s.EntryByID(ctx, 1, list[1].ID)
	if err != nil || own != list[1] {
		t.Fatal(own, err)
	}
	for _, id := range []int64{list[0].ID, 999} {
		if _, err := s.EntryByID(ctx, 2, id); !errors.Is(err, store.ErrNotFound) {
			t.Fatal(err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.EntryByID(canceled, 1, list[0].ID); err == nil {
		t.Fatal("missing cancellation")
	}
	if _, err := s.ListEntries(canceled, 1); err == nil {
		t.Fatal("missing cancellation")
	}
}
func TestBookConstraints(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := db.Exec(`INSERT INTO users(id,name,phone) VALUES(1,'a','1'); INSERT INTO addresses(id,city,address) VALUES(1,'Город','Дом'),(2,'Город','Дом 2'); INSERT INTO user_addresses(user_id,address_id,is_default) VALUES(1,1,true)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_addresses(user_id,address_id,is_default) VALUES(1,2,true)`); !store.IsUniqueViolation(err) {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_addresses(user_id,address_id) VALUES(1,1)`); !store.IsUniqueViolation(err) {
		t.Fatal(err)
	}
	for _, column := range []string{"name", "street", "apartment", "entrance"} {
		if _, err := db.Exec(`UPDATE user_addresses SET `+column+`=$1`, strings.Repeat("я", 121)); !store.IsCheckViolation(err) {
			t.Fatal(column, err)
		}
	}
	if _, err := db.Exec(`UPDATE user_addresses SET street='я'`); !store.IsCheckViolation(err) {
		t.Fatal(err)
	}
	var full string
	if err := db.Get(&full, `SELECT address FROM addresses WHERE id=1`); err != nil || full != "Дом" {
		t.Fatal(full, err)
	}
}
