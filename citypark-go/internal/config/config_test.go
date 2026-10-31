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
		{"CustomerAuthEnabled", c.CustomerAuthEnabled, false},
		{"CustomerPublicOrigin", c.CustomerPublicOrigin, "http://localhost:3001"},
		{"SMTP", c.SMTP, SMTP{Port: 587, TLSMode: "starttls"}},
		{"Admin", c.Admin, Admin{"dev@dev.io", "devdevdev", "+79990000000"}},
		{"Iiko", c.Iiko, Iiko{BaseURL: "https://api-ru.iiko.services/api", PaymentTypeKind: "Card", AddressCity: "Белореченск", AddressStreet: "Гоголя", AddressHouse: "61"}},
		{"Telegram", c.Telegram, Telegram{BaseURL: "https://api.telegram.org"}},
		{"Moneta", c.Moneta, Moneta{URL: "https://demo.moneta.ru/assistant.htm", SuccessURL: "http://localhost:3001/checkout/done", FailURL: "http://localhost:3001/checkout?payment=failed"}},
		{"Hotel", c.Hotel, Hotel{}},
		{"OneC", c.OneC, OneC{}},
		{"Upload", c.Upload, Upload{"./uploads", "http://localhost:3000"}},
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

func TestGoogleConfig(t *testing.T) {
	if c, err := Load(lookupMap(nil)); err != nil || c.Google.Enabled {
		t.Fatalf("absent: %+v %v", c.Google, err)
	}
	valid := map[string]string{"CUSTOMER_PUBLIC_ORIGIN": "https://site.example", "GOOGLE_CLIENT_ID": "client", "GOOGLE_CLIENT_SECRET": "secret", "GOOGLE_CALLBACK_URL": "https://site.example/account/google/callback"}
	if c, err := Load(lookupMap(valid)); err != nil || !c.Google.Enabled {
		t.Fatalf("valid: %+v %v", c.Google, err)
	}
	for _, key := range []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_CALLBACK_URL"} {
		copy := map[string]string{}
		for k, v := range valid {
			copy[k] = v
		}
		delete(copy, key)
		if _, err := Load(lookupMap(copy)); err == nil {
			t.Errorf("missing %s accepted", key)
		}
	}
	for _, callback := range []string{"http://site.example/account/google/callback", "https://other.example/account/google/callback", "https://site.example/account/google/other", "https://site.example/account/google/callback?next=x"} {
		copy := map[string]string{}
		for k, v := range valid {
			copy[k] = v
		}
		copy["GOOGLE_CALLBACK_URL"] = callback
		if _, err := Load(lookupMap(copy)); err == nil {
			t.Errorf("unsafe callback accepted: %s", callback)
		}
	}
}

func TestCustomerTelegramBotConfig(t *testing.T) {
	c, err := Load(lookupMap(map[string]string{"TELEGRAM_TOKEN": "staff", "CUSTOMER_TELEGRAM_BOT_TOKEN": "customer"}))
	if err != nil || c.Telegram.Token != "staff" || c.CustomerTelegramBotToken != "customer" {
		t.Fatalf("separate bot configuration: %+v %v", c, err)
	}
	c, err = Load(lookupMap(map[string]string{"TELEGRAM_TOKEN": "staff"}))
	if err != nil || c.CustomerTelegramBotToken != "" {
		t.Fatalf("staff token used as fallback: %+v %v", c, err)
	}
}

func TestLoadOverrides(t *testing.T) {
	c, err := Load(lookupMap(map[string]string{
		"APP_ENV":                "test",
		"PORT":                   "8080",
		"DEBUG":                  "true",
		"APP_TIMEZONE":           "Asia/Yekaterinburg",
		"DEFAULT_PAGE_SIZE":      "20",
		"DATABASE_URL":           "postgres://x",
		"JWT_SECRET":             "s3cret",
		"JWT_EXPIRES_IN":         "90m",
		"CUSTOMER_AUTH_ENABLED":  "true",
		"CUSTOMER_PUBLIC_ORIGIN": "https://site.example",
		"SMTP_HOST":              "smtp.example",
		"SMTP_USERNAME":          "user",
		"SMTP_PASSWORD":          "secret",
		"SMTP_FROM":              "mail@example.com",
		"ADMIN_EMAIL":            "a@b.c",
		"ADMIN_PASSWORD":         "pw",
		"ADMIN_PHONE":            "+70000000000",
		"IIKO_BASE_URL":          "http://iiko",
		"IIKO_API_KEY":           "key",
		"IIKO_APP_ID":            "app",
		"IIKO_CLIENT_SECRET":     "secret",
		"IIKO_ORGANIZATION_ID":   "org",
		"IIKO_EXTERNAL_MENU_ID":  "menu",
		"IIKO_UPDATE_TOKEN":      "tok",
		"IIKO_SYNC_INTERVAL":     "30m",
		"IIKO_TERMINAL_GROUP_ID": "tg1",
		"IIKO_PAYMENT_TYPE_ID":   "pt1",
		"IIKO_PAYMENT_TYPE_KIND": "External",
		"IIKO_ADDRESS_CITY":      "Майкоп",
		"IIKO_ADDRESS_STREET":    "Мира",
		"IIKO_ADDRESS_HOUSE":     "1",
		"TELEGRAM_BASE_URL":      "http://tg",
		"TELEGRAM_TOKEN":         "tgtok",
		"TELEGRAM_STAFF_CHAT_ID": "2",
		"MONETA_ID":              "100",
		"MONETA_SECRET":          "s",
		"MONETA_TEST_MODE":       "true",
		"MONETA_URL":             "https://pay.example/",
		"MONETA_SUCCESS_URL":     "https://site/ok",
		"MONETA_FAIL_URL":        "https://site/fail",
		"HOTEL_WEBHOOK_SECRET":   "hotel",
		"ONEC_HOTEL_URL":         "http://1c.local/CityParkHotel/ru_RU",
		"ONEC_USER":              "reports",
		"ONEC_PASSWORD":          "pw1c",
		"UPLOAD_DIR":             "/data/uploads",
		"PUBLIC_BASE_URL":        "https://api.example.com/",
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
		{"CustomerAuthEnabled", c.CustomerAuthEnabled, true},
		{"CustomerPublicOrigin", c.CustomerPublicOrigin, "https://site.example"},
		{"SMTP", c.SMTP, SMTP{Host: "smtp.example", Port: 587, Username: "user", Password: "secret", From: "mail@example.com", TLSMode: "starttls", Enabled: true}},
		{"Admin", c.Admin, Admin{"a@b.c", "pw", "+70000000000"}},
		{"Iiko", c.Iiko, Iiko{"", "http://iiko", "key", "app", "secret", "org", "menu", "tok", 30 * time.Minute, "tg1", "pt1", "External", "Майкоп", "Мира", "1"}},
		{"Telegram", c.Telegram, Telegram{"http://tg", "tgtok", "2"}},
		{"Moneta", c.Moneta, Moneta{"100", "s", true, "https://pay.example/", "https://site/ok", "https://site/fail"}},
		{"Hotel", c.Hotel, Hotel{"hotel"}},
		{"OneC", c.OneC, OneC{"http://1c.local/CityParkHotel/ru_RU", "reports", "pw1c"}},
		{"Upload", c.Upload, Upload{"/data/uploads", "https://api.example.com"}},
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
	_, err := Load(lookupMap(map[string]string{"APP_ENV": "production", "CUSTOMER_PUBLIC_ORIGIN": "https://site.example"}))
	if err == nil || err.Error() != "JWT_SECRET must be changed in production" {
		t.Fatalf("err = %v, want JWT_SECRET must be changed in production", err)
	}

	c, err := Load(lookupMap(map[string]string{"APP_ENV": "production", "JWT_SECRET": "x", "CUSTOMER_PUBLIC_ORIGIN": "https://site.example"}))
	if err != nil {
		t.Fatal(err)
	}
	if !c.IsProduction() || c.IsTest() {
		t.Error("IsProduction/IsTest wrong for env production")
	}
}

func TestSMTPConfig(t *testing.T) {
	valid := map[string]string{"CUSTOMER_AUTH_ENABLED": "true", "CUSTOMER_PUBLIC_ORIGIN": "https://site.example", "SMTP_HOST": "smtp.example", "SMTP_USERNAME": "user", "SMTP_PASSWORD": "secret", "SMTP_FROM": "from@example.com"}
	copyWith := func(k, v string) map[string]string {
		m := make(map[string]string, len(valid)+1)
		for key, value := range valid {
			m[key] = value
		}
		m[k] = v
		return m
	}
	if c, err := Load(lookupMap(valid)); err != nil || !c.SMTP.Enabled || c.SMTP.Port != 587 || c.SMTP.TLSMode != "starttls" {
		t.Fatalf("valid: %+v %v", c.SMTP, err)
	}
	if c, err := Load(lookupMap(copyWith("SMTP_TLS_MODE", "tls"))); err != nil || c.SMTP.TLSMode != "tls" {
		t.Fatalf("tls: %+v %v", c.SMTP, err)
	}
	if c, err := Load(lookupMap(map[string]string{"SMTP_HOST": "smtp.example"})); err != nil || c.SMTP.Enabled {
		t.Fatalf("disabled partial: %+v %v", c.SMTP, err)
	}
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"missing host", copyWith("SMTP_HOST", ""), "SMTP configuration"}, {"missing user", copyWith("SMTP_USERNAME", ""), "SMTP configuration"}, {"missing password", copyWith("SMTP_PASSWORD", ""), "SMTP configuration"}, {"missing from", copyWith("SMTP_FROM", ""), "SMTP configuration"},
		{"bad port", copyWith("SMTP_PORT", "65536"), "SMTP_PORT"}, {"bad mode", copyWith("SMTP_TLS_MODE", "plain"), "SMTP_TLS_MODE"}, {"bad sender", copyWith("SMTP_FROM", "Name <from@example.com>"), "SMTP_FROM"}, {"sender newline", copyWith("SMTP_FROM", "from@example.com\r\nBcc: victim@example.com"), "SMTP"},
		{"origin path", copyWith("CUSTOMER_PUBLIC_ORIGIN", "https://site.example/path"), "CUSTOMER_PUBLIC_ORIGIN"}, {"origin query", copyWith("CUSTOMER_PUBLIC_ORIGIN", "https://site.example?x=1"), "CUSTOMER_PUBLIC_ORIGIN"}, {"origin userinfo", copyWith("CUSTOMER_PUBLIC_ORIGIN", "https://user@site.example"), "CUSTOMER_PUBLIC_ORIGIN"}, {"unsafe http", copyWith("CUSTOMER_PUBLIC_ORIGIN", "http://site.example"), "CUSTOMER_PUBLIC_ORIGIN"}, {"production http", func() map[string]string {
			m := copyWith("APP_ENV", "production")
			m["CUSTOMER_PUBLIC_ORIGIN"] = "http://localhost:3001"
			m["JWT_SECRET"] = "safe"
			return m
		}(), "CUSTOMER_PUBLIC_ORIGIN"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Load(lookupMap(tc.env)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %s", err, tc.want)
			}
		})
	}
	if c, err := Load(lookupMap(map[string]string{"CUSTOMER_PUBLIC_ORIGIN": "http://127.0.0.1:3001"})); err != nil || c.CustomerPublicOrigin != "http://127.0.0.1:3001" {
		t.Fatalf("localhost: %+v %v", c, err)
	}
}

func TestPromoConfiguration(t *testing.T) {
	for _, id := range []string{"", "01234567-89ab-4def-8123-456789abcdef", "01234567-89AB-4DEF-8123-456789ABCDEF", "bad", "00000000-0000-0000-0000-000000000000", "01234567-89ab-4def-8123-456789abcdefx"} {
		c, err := Load(lookupMap(map[string]string{"IIKO_PROMO_DISCOUNT_TYPE_ID": id}))
		valid := id == "" || id == "01234567-89ab-4def-8123-456789abcdef" || id == "01234567-89AB-4DEF-8123-456789ABCDEF"
		if (err == nil) != valid {
			t.Fatalf("%q: %v", id, err)
		}
		if valid && c.Iiko.PromoDiscountTypeID != id {
			t.Fatal(c.Iiko)
		}
		if !valid && !strings.Contains(err.Error(), "IIKO_PROMO_DISCOUNT_TYPE_ID") {
			t.Fatal(err)
		}
	}
}
