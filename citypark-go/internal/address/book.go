package address

import (
	"encoding/json"
	"golang.org/x/text/unicode/norm"
	"strings"
	"time"
	"unicode/utf8"
)

type Fields struct {
	Name      string `json:"name" db:"name" validate:"max=40"`
	City      string `json:"city" db:"city" validate:"required,max=80"`
	Street    string `json:"street" db:"street" validate:"required,min=3,max=120"`
	Apartment string `json:"apartment" db:"apartment" validate:"max=20"`
	Entrance  string `json:"entrance" db:"entrance" validate:"max=20"`
}

type Entry struct {
	ID          int64     `json:"id" db:"id"`
	AddressID   int64     `json:"-" db:"address_id"`
	Name        string    `json:"name" db:"name"`
	City        string    `json:"city" db:"city"`
	Address     string    `json:"address" db:"address"`
	Street      string    `json:"street" db:"street"`
	Apartment   string    `json:"apartment" db:"apartment"`
	Entrance    string    `json:"entrance" db:"entrance"`
	IsDefault   bool      `json:"isDefault" db:"is_default"`
	NeedsReview bool      `json:"needsReview" db:"needs_review"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

func CleanFields(in Fields) Fields {
	in.Name = strings.TrimSpace(in.Name)
	in.City = strings.TrimSpace(in.City)
	in.Street = strings.TrimSpace(in.Street)
	in.Apartment = strings.TrimSpace(in.Apartment)
	in.Entrance = strings.TrimSpace(in.Entrance)
	return in
}

func FieldErrors(in Fields) map[string]string {
	errors := map[string]string{}
	for _, f := range []struct {
		key, value    string
		min, max      int
		missing, long string
	}{
		{"name", in.Name, 0, 40, "", "Не больше 40 символов"},
		{"city", in.City, 1, 80, "Выберите населённый пункт", "Не больше 80 символов"},
		{"street", in.Street, 3, 120, "Укажите улицу и дом", "Слишком длинный адрес"},
		{"apartment", in.Apartment, 0, 20, "", "Не больше 20 символов"},
		{"entrance", in.Entrance, 0, 20, "", "Не больше 20 символов"},
	} {
		n := utf8.RuneCountInString(f.value)
		if n < f.min {
			errors[f.key] = f.missing
		} else if n > f.max {
			errors[f.key] = f.long
		}
	}
	return errors
}

func FormatAddress(in Fields) string {
	parts := []string{}
	if in.Street != "" {
		parts = append(parts, in.Street)
	}
	if in.Apartment != "" {
		parts = append(parts, "кв. "+in.Apartment)
	}
	if in.Entrance != "" {
		parts = append(parts, "подъезд "+in.Entrance)
	}
	return strings.Join(parts, ", ")
}

func DuplicateKey(city, fullAddress string) string {
	normalize := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(norm.NFC.String(s)), " ")) }
	data, _ := json.Marshal([2]string{normalize(city), normalize(fullAddress)})
	return string(data)
}
