package validate

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

type item struct {
	ProductID int64 `json:"productId" validate:"required,gt=0"`
	Amount    int   `json:"amount" validate:"gt=0"`
}

type addr struct {
	City string `json:"city" validate:"required,max=80"`
}

type req struct {
	Name    string `json:"name" validate:"required,max=80"`
	Email   string `json:"email" validate:"omitempty,email"`
	Phone   string `json:"phone" validate:"required,ruphone"`
	Kind    string `json:"kind" validate:"oneof=cash online"`
	Address *addr  `json:"address" validate:"omitempty"`
	Items   []item `json:"items" validate:"required,min=1,dive"`
}

func validReq() req {
	return req{
		Name:  "Иван",
		Email: "ivan@example.com",
		Phone: "+79991234567",
		Kind:  "cash",
		Items: []item{{ProductID: 1, Amount: 2}},
	}
}

func asHTTPError(t *testing.T, err error) *httpx.Error {
	t.Helper()
	var e *httpx.Error
	if !errors.As(err, &e) {
		t.Fatalf("error = %T (%v), want *httpx.Error", err, err)
	}
	return e
}

func TestStructValid(t *testing.T) {
	if err := Struct(validReq()); err != nil {
		t.Fatalf("Struct() = %v, want nil", err)
	}
}

func TestStructOptionalEmail(t *testing.T) {
	r := validReq()
	r.Email = ""
	if err := Struct(r); err != nil {
		t.Fatalf("Struct() = %v, want nil", err)
	}
}

func TestStructMessages(t *testing.T) {
	r := req{
		Name:    "",
		Email:   "not-an-email",
		Phone:   "123",
		Kind:    "card",
		Address: &addr{City: ""},
		Items:   []item{{ProductID: 1, Amount: 0}},
	}
	e := asHTTPError(t, Struct(r))

	if e.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want 400", e.Status)
	}
	const wantTop = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
	if e.Message != wantTop {
		t.Errorf("Message = %q, want %q", e.Message, wantTop)
	}
	want := map[string]string{
		"name":            "поле не должно быть пустым",
		"email":           "email должен быть электронной почтой",
		"phone":           "телефон должен быть действительным номером телефона",
		"kind":            "недопустимое значение",
		"address.city":    "поле не должно быть пустым",
		"items[0].amount": "значение должно быть больше 0",
	}
	if !reflect.DeepEqual(e.Fields, want) {
		t.Errorf("Fields = %v, want %v", e.Fields, want)
	}
}

func TestStructParamMessages(t *testing.T) {
	type s struct {
		Long  string `json:"long" validate:"max=3"`
		Short string `json:"short" validate:"min=5"`
		Count int    `json:"count" validate:"gte=2"`
		URL   string `json:"url" validate:"url"`
	}
	e := asHTTPError(t, Struct(s{Long: "abcdef", Short: "ab", Count: 1, URL: "nope"}))
	want := map[string]string{
		"long":  "значение должно быть не больше 3",
		"short": "значение должно быть не меньше 5",
		"count": "значение должно быть не меньше 2",
		"url":   "некорректное значение",
	}
	if !reflect.DeepEqual(e.Fields, want) {
		t.Errorf("Fields = %v, want %v", e.Fields, want)
	}
}

func TestStructFieldNames(t *testing.T) {
	type s struct {
		WithOptions string `json:"withOptions,omitempty" validate:"required"`
		Skipped     string `json:"-" validate:"required"`
		NoTag       string `validate:"required"`
	}
	e := asHTTPError(t, Struct(s{}))
	want := map[string]string{
		"withOptions": "поле не должно быть пустым",
		"Skipped":     "поле не должно быть пустым",
		"NoTag":       "поле не должно быть пустым",
	}
	if !reflect.DeepEqual(e.Fields, want) {
		t.Errorf("Fields = %v, want %v", e.Fields, want)
	}
}

func TestStructNonValidationError(t *testing.T) {
	err := Struct(42)
	if err == nil {
		t.Fatal("Struct(42) = nil, want error")
	}
	var he *httpx.Error
	if errors.As(err, &he) {
		t.Fatalf("error is *httpx.Error (%v), want it returned unchanged", he)
	}
	var inv *validator.InvalidValidationError
	if !errors.As(err, &inv) {
		t.Errorf("error = %T, want *validator.InvalidValidationError", err)
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		want   string
		wantOK bool
	}{
		{"plus seven", "+79991234567", "+79991234567", true},
		{"eight with separators", "8 (999) 123-45-67", "+79991234567", true},
		{"seven without plus", "79991234567", "+79991234567", true},
		{"no country code", "9991234567", "", false},
		{"too short", "+7999123456", "", false},
		{"too long", "+799912345678", "", false},
		{"letter", "+7999123456a", "", false},
		{"empty", "", "", false},
		{"other country", "+19991234567", "", false},
		{"only prefix", "+7", "", false},
		{"non-ascii digit", "+7999123456" + strings.Repeat("٣", 1), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizePhone(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("NormalizePhone(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
