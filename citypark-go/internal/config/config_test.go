package config

import (
	"strings"
	"testing"
	"time"
)

func lookupMap(m map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func TestLoadDefaults(t *testing.T) {
	c, err := Load(lookupMap(nil))
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"Env", c.Env, "development"},
		{"Port", c.Port, 3000},
		{"Debug", c.Debug, false},
		{"Timezone", c.Timezone.String(), "Europe/Moscow"},
		{"DefaultPageSize", c.DefaultPageSize, 100},
		{"DatabaseURL", c.DatabaseURL, "postgres://postgres:postgres@localhost:5432/citypark_dev?sslmode=disable"},
		{"JWT.Secret", c.JWT.Secret, "changeme"},
		{"JWT.ExpiresIn", c.JWT.ExpiresIn, 24 * time.Hour},
		{"Admin", c.Admin, Admin{"dev@dev.io", "devdevdev", "+79990000000"}},
		{"Iiko", c.Iiko, Iiko{BaseURL: "https://api-ru.iiko.services/api"}},
		{"Telegram", c.Telegram, Telegram{BaseURL: "https://api.telegram.org"}},
		{"Moneta", c.Moneta, Moneta{}},
		{"Hotel", c.Hotel, Hotel{}},
		{"Upload", c.Upload, Upload{"./uploads", "http://localhost:3000"}},
		{"WorkingHours", c.WorkingHours, WorkingHours{"9:00", "22:00"}},
		{"ThankYouPageCash", c.ThankYouPageCash, "http://localhost:3001/thankyou"},
	}
	for _, ch := range checks {
		if ch.got != ch.want {
			t.Errorf("%s = %v, want %v", ch.name, ch.got, ch.want)
		}
	}
	if c.IsProduction() || c.IsTest() {
		t.Error("development must be neither production nor test")
	}
}

func TestLoadOverrides(t *testing.T) {
	c, err := Load(lookupMap(map[string]string{
		"APP_ENV":                              "test",
		"PORT":                                 "8080",
		"DEBUG":                                "true",
		"APP_TIMEZONE":                         "Asia/Yekaterinburg",
		"DEFAULT_PAGE_SIZE":                    "20",
		"DATABASE_URL":                         "postgres://x",
		"JWT_SECRET":                           "s3cret",
		"JWT_EXPIRES_IN":                       "90m",
		"ADMIN_EMAIL":                          "a@b.c",
		"ADMIN_PASSWORD":                       "pw",
		"ADMIN_PHONE":                          "+70000000000",
		"IIKO_BASE_URL":                        "http://iiko",
		"IIKO_API_KEY":                         "key",
		"IIKO_APP_ID":                          "app",
		"IIKO_CLIENT_SECRET":                   "secret",
		"IIKO_ORGANIZATION_ID":                 "org",
		"IIKO_EXTERNAL_MENU_ID":                "menu",
		"IIKO_UPDATE_TOKEN":                    "tok",
		"IIKO_SYNC_INTERVAL":                   "30m",
		"TELEGRAM_BASE_URL":                    "http://tg",
		"TELEGRAM_TOKEN":                       "tgtok",
		"TELEGRAM_CHAT_ID_FOR_ORDER_CREATED":   "1",
		"TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED": "2",
		"MONETA_ID":                            "100",
		"MONETA_SECRET":                        "s",
		"MONETA_TEST_MODE":                     "true",
		"HOTEL_WEBHOOK_SECRET":                 "hotel",
		"UPLOAD_DIR":                           "/data/uploads",
		"PUBLIC_BASE_URL":                      "https://api.example.com/",
		"WORKING_HOURS_START":                  "10:30",
		"WORKING_HOURS_END":                    "23:00",
		"THANK_YOU_PAGE_PAYMENT_CASH":          "http://cash",
	}))
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"Env", c.Env, "test"},
		{"Port", c.Port, 8080},
		{"Debug", c.Debug, true},
		{"Timezone", c.Timezone.String(), "Asia/Yekaterinburg"},
		{"DefaultPageSize", c.DefaultPageSize, 20},
		{"DatabaseURL", c.DatabaseURL, "postgres://x"},
		{"JWT", c.JWT, JWT{"s3cret", 90 * time.Minute}},
		{"Admin", c.Admin, Admin{"a@b.c", "pw", "+70000000000"}},
		{"Iiko", c.Iiko, Iiko{"http://iiko", "key", "app", "secret", "org", "menu", "tok", 30 * time.Minute}},
		{"Telegram", c.Telegram, Telegram{"http://tg", "tgtok", "1", "2"}},
		{"Moneta", c.Moneta, Moneta{"100", "s", true}},
		{"Hotel", c.Hotel, Hotel{"hotel"}},
		{"Upload", c.Upload, Upload{"/data/uploads", "https://api.example.com"}},
		{"WorkingHours", c.WorkingHours, WorkingHours{"10:30", "23:00"}},
		{"ThankYouPageCash", c.ThankYouPageCash, "http://cash"},
	}
	for _, ch := range checks {
		if ch.got != ch.want {
			t.Errorf("%s = %v, want %v", ch.name, ch.got, ch.want)
		}
	}
	if !c.IsTest() || c.IsProduction() {
		t.Error("IsTest/IsProduction wrong for env test")
	}
}

func TestLoadDebugOnlyTrue(t *testing.T) {
	for v, want := range map[string]bool{"true": true, "1": false, "TRUE": false, "": false} {
		c, err := Load(lookupMap(map[string]string{"DEBUG": v}))
		if err != nil {
			t.Fatal(err)
		}
		if c.Debug != want {
			t.Errorf("DEBUG=%q: Debug = %v, want %v", v, c.Debug, want)
		}
	}
}

func TestLoadMonetaTestMode(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"not set", nil, false},
		{"true", map[string]string{"MONETA_TEST_MODE": "true"}, true},
		{"1", map[string]string{"MONETA_TEST_MODE": "1"}, false},
		{"empty", map[string]string{"MONETA_TEST_MODE": ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Load(lookupMap(tt.env))
			if err != nil {
				t.Fatal(err)
			}
			if c.Moneta.TestMode != tt.want {
				t.Errorf("TestMode = %v, want %v", c.Moneta.TestMode, tt.want)
			}
		})
	}
}

func TestLoadNoOldVariables(t *testing.T) {
	var asked []string
	_, err := Load(func(key string) (string, bool) {
		asked = append(asked, key)
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) == 0 {
		t.Fatal("Load did not ask lookup for anything")
	}
	for _, k := range asked {
		if strings.HasPrefix(k, "YOOKASSA_") || k == "THANK_YOU_PAGE_PAYMENT_ONLINE" {
			t.Errorf("Load asked for removed variable %s", k)
		}
	}
}

func TestLoadEmptyWorkingHours(t *testing.T) {
	c, err := Load(lookupMap(map[string]string{"WORKING_HOURS_START": "", "WORKING_HOURS_END": ""}))
	if err != nil {
		t.Fatal(err)
	}
	if c.WorkingHours.Start != "" || c.WorkingHours.End != "" {
		t.Errorf("WorkingHours = %+v, want both empty", c.WorkingHours)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"port not a number", map[string]string{"PORT": "abc"}, "PORT"},
		{"port zero", map[string]string{"PORT": "0"}, "PORT"},
		{"port negative", map[string]string{"PORT": "-5"}, "PORT"},
		{"port empty", map[string]string{"PORT": ""}, "PORT"},
		{"page size bad", map[string]string{"DEFAULT_PAGE_SIZE": "0"}, "DEFAULT_PAGE_SIZE"},
		{"bad duration", map[string]string{"JWT_EXPIRES_IN": "tomorrow"}, "JWT_EXPIRES_IN"},
		{"bad timezone", map[string]string{"APP_TIMEZONE": "Mars/Base"}, "APP_TIMEZONE"},
		{"bad env", map[string]string{"APP_ENV": "staging"}, "APP_ENV"},
		{"sync interval negative", map[string]string{"IIKO_SYNC_INTERVAL": "-1m"}, "IIKO_SYNC_INTERVAL is invalid"},
		{"sync interval garbage", map[string]string{"IIKO_SYNC_INTERVAL": "abc"}, "IIKO_SYNC_INTERVAL is invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(lookupMap(tt.env))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not name %s", err, tt.wantErr)
			}
		})
	}
}

func TestLoadProductionSecret(t *testing.T) {
	_, err := Load(lookupMap(map[string]string{"APP_ENV": "production"}))
	if err == nil || err.Error() != "JWT_SECRET must be changed in production" {
		t.Fatalf("err = %v, want JWT_SECRET must be changed in production", err)
	}

	c, err := Load(lookupMap(map[string]string{"APP_ENV": "production", "JWT_SECRET": "x"}))
	if err != nil {
		t.Fatal(err)
	}
	if !c.IsProduction() || c.IsTest() {
		t.Error("IsProduction/IsTest wrong for env production")
	}
}
