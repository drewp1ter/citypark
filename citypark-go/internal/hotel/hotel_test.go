package hotel

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestStoreGetEmpty(t *testing.T) {
	s := NewStore(testutil.NewDB(t))

	_, err := s.Get(context.Background())

	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want store.ErrNotFound", err)
	}
}

func TestStoreSetThenGet(t *testing.T) {
	s := NewStore(testutil.NewDB(t))
	want := Rooms{VIP: 1, SingleBalcony: 0, Single: 2, DoubleTwin: 1, DoubleEuro: 0}

	if _, err := s.Set(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if got.Rooms != want {
		t.Errorf("rooms = %+v, want %+v", got.Rooms, want)
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestStoreSetTwice(t *testing.T) {
	db := testutil.NewDB(t)
	s := NewStore(db)
	second := Rooms{VIP: 0, SingleBalcony: 1, Single: 1, DoubleTwin: 4, DoubleEuro: 3}

	if _, err := s.Set(context.Background(), Rooms{VIP: 1, Single: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Set(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	if n := testutil.Count(t, db, `SELECT count(*) FROM hotel_rooms`); n != 1 {
		t.Errorf("rows = %d, want 1", n)
	}
	got, err := s.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Rooms != second {
		t.Errorf("rooms = %+v, want %+v", got.Rooms, second)
	}
}

func TestRoomsFree(t *testing.T) {
	tests := []struct {
		name  string
		rooms Rooms
		want  int
	}{
		{"all free", Rooms{1, 1, 2, 4, 4}, 12},
		{"none free", Rooms{}, 0},
		{"mixed", Rooms{1, 0, 2, 1, 0}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rooms.Free(); got != tt.want {
				t.Errorf("Free() = %d, want %d", got, tt.want)
			}
		})
	}
}

const validBody = `{"rooms": {"vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0}}`

func newRouter(db *sqlx.DB, secret string) http.Handler {
	r := chi.NewRouter()
	NewHandler(NewStore(db), secret, auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Now()))).Routes(r)
	return r
}

// post sends POST /hotel/rooms; header "" sends no X-Webhook-Secret.
func post(h http.Handler, header, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/hotel/rooms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set("X-Webhook-Secret", header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type roomsBody struct {
	Rooms     Rooms  `json:"rooms"`
	Free      int    `json:"free"`
	UpdatedAt string `json:"updatedAt"`
}

type errorBody struct {
	StatusCode int               `json:"statusCode"`
	Message    string            `json:"message"`
	Errors     map[string]string `json:"errors"`
}

func TestGetRoomsEmpty(t *testing.T) {
	h := newRouter(testutil.NewDB(t), "s")

	rec := testutil.DoJSON(t, h, http.MethodGet, "/hotel/rooms", "", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var e errorBody
	testutil.Decode(t, rec, &e)
	if e.Message != "Не найдено" {
		t.Errorf("message = %q, want %q", e.Message, "Не найдено")
	}
}

func TestGetRooms(t *testing.T) {
	db := testutil.NewDB(t)
	want := Rooms{VIP: 1, SingleBalcony: 0, Single: 2, DoubleTwin: 1, DoubleEuro: 0}
	if _, err := NewStore(db).Set(context.Background(), want); err != nil {
		t.Fatal(err)
	}

	rec := testutil.DoJSON(t, newRouter(db, "s"), http.MethodGet, "/hotel/rooms", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	var got roomsBody
	testutil.Decode(t, rec, &got)
	if got.Rooms != want || got.Free != 4 || got.UpdatedAt == "" {
		t.Errorf("body = %+v, want rooms %+v, free 4 and updatedAt set", got, want)
	}
}

func TestPostDisabled(t *testing.T) {
	db := testutil.NewDB(t)

	// With no secret configured the endpoint does not exist, whatever the header says.
	rec := post(newRouter(db, ""), "anything", validBody)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM hotel_rooms`); n != 0 {
		t.Errorf("rows = %d, want 0", n)
	}
}

func TestPostUnauthorized(t *testing.T) {
	for _, tt := range []struct{ name, header string }{
		{"no header", ""},
		{"wrong header", "other"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewDB(t)

			rec := post(newRouter(db, "secret"), tt.header, validBody)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			var e errorBody
			testutil.Decode(t, rec, &e)
			if e.Message != "Требуется авторизация" {
				t.Errorf("message = %q, want %q", e.Message, "Требуется авторизация")
			}
			if n := testutil.Count(t, db, `SELECT count(*) FROM hotel_rooms`); n != 0 {
				t.Errorf("rows = %d, want 0", n)
			}
		})
	}
}

func TestPostInvalid(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		field string
		msg   string
	}{
		{"missing doubleEuro", `{"rooms": {"vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1}}`,
			"rooms.doubleEuro", "поле не должно быть пустым"},
		{"vip too big", `{"rooms": {"vip": 2, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0}}`,
			"rooms.vip", "значение должно быть не больше 1"},
		{"singleBalcony too big", `{"rooms": {"vip": 1, "singleBalcony": 2, "single": 2, "doubleTwin": 1, "doubleEuro": 0}}`,
			"rooms.singleBalcony", "значение должно быть не больше 1"},
		{"single too big", `{"rooms": {"vip": 1, "singleBalcony": 0, "single": 3, "doubleTwin": 1, "doubleEuro": 0}}`,
			"rooms.single", "значение должно быть не больше 2"},
		{"doubleTwin too big", `{"rooms": {"vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 5, "doubleEuro": 0}}`,
			"rooms.doubleTwin", "значение должно быть не больше 4"},
		{"doubleEuro negative", `{"rooms": {"vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": -1}}`,
			"rooms.doubleEuro", "значение должно быть не меньше 0"},
		{"no rooms", `{}`, "rooms", "поле не должно быть пустым"},
		{"old body", `{"rooms": {"double_rooms": 9, "single_rooms": 2, "vip_room": 1}}`,
			"rooms.vip", "поле не должно быть пустым"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewDB(t)

			rec := post(newRouter(db, "secret"), "secret", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body)
			}
			var e errorBody
			testutil.Decode(t, rec, &e)
			if got := e.Errors[tt.field]; got != tt.msg {
				t.Errorf("errors[%q] = %q, want %q (all: %v)", tt.field, got, tt.msg, e.Errors)
			}
			if n := testutil.Count(t, db, `SELECT count(*) FROM hotel_rooms`); n != 0 {
				t.Errorf("rows = %d, want 0", n)
			}
		})
	}
}

func TestPostMalformedJSON(t *testing.T) {
	db := testutil.NewDB(t)

	rec := post(newRouter(db, "secret"), "secret", `{"rooms":`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM hotel_rooms`); n != 0 {
		t.Errorf("rows = %d, want 0", n)
	}
}

func TestPostValid(t *testing.T) {
	db := testutil.NewDB(t)

	rec := post(newRouter(db, "secret"), "secret", validBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	var got roomsBody
	testutil.Decode(t, rec, &got)
	want := Rooms{VIP: 1, SingleBalcony: 0, Single: 2, DoubleTwin: 1, DoubleEuro: 0}
	if got.Rooms != want || got.Free != 4 {
		t.Errorf("body = %+v, want rooms %+v and free 4", got, want)
	}
	st, err := NewStore(db).Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Rooms != want {
		t.Errorf("stored = %+v, want %+v", st.Rooms, want)
	}
}

func TestPostMax(t *testing.T) {
	body := `{"rooms": {"vip": 1, "singleBalcony": 1, "single": 2, "doubleTwin": 4, "doubleEuro": 4}}`

	rec := post(newRouter(testutil.NewDB(t), "secret"), "secret", body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	var got roomsBody
	testutil.Decode(t, rec, &got)
	if got.Free != 12 {
		t.Errorf("free = %d, want 12", got.Free)
	}
}

func TestPostZero(t *testing.T) {
	body := `{"rooms": {"vip": 0, "singleBalcony": 0, "single": 0, "doubleTwin": 0, "doubleEuro": 0}}`

	rec := post(newRouter(testutil.NewDB(t), "secret"), "secret", body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	var got roomsBody
	testutil.Decode(t, rec, &got)
	if got.Free != 0 {
		t.Errorf("free = %d, want 0", got.Free)
	}
}
