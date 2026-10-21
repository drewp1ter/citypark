// Package config reads application settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

type Config struct {
	Env              string // "development", "production" or "test"
	Port             int
	Debug            bool
	Timezone         *time.Location
	DefaultPageSize  int
	DatabaseURL      string
	JWT              JWT
	Admin            Admin
	Iiko             Iiko
	Telegram         Telegram
	Moneta           Moneta
	WorkingHours     WorkingHours
	ThankYouPageCash string
}

type JWT struct {
	Secret    string
	ExpiresIn time.Duration
}
type Admin struct{ Email, Password, Phone string }
type Iiko struct{ BaseURL, APIKey, AppID, ClientSecret, OrganizationID, ExternalMenuID, UpdateToken string }
type Telegram struct{ BaseURL, Token, ChatIDOrderCreated, ChatIDOrderConfirmed string }
type Moneta struct {
	ID       string // shop id, sent to Moneta as MNT_ID
	Secret   string // integrity code from the Moneta cabinet
	TestMode bool
}
type WorkingHours struct{ Start, End string } // "9:00"; empty string means no restriction

// LookupFunc has the signature of os.LookupEnv.
type LookupFunc func(key string) (string, bool)

const defaultJWTSecret = "changeme"

// Load reads the configuration. Pass os.LookupEnv in production code.
func Load(lookup LookupFunc) (Config, error) {
	str := func(key, def string) string {
		if v, ok := lookup(key); ok {
			return v
		}
		return def
	}
	positiveInt := func(key, def string) (int, error) {
		v := str(key, def)
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("%s must be a positive integer, got %q", key, v)
		}
		return n, nil
	}

	var c Config
	var err error

	c.Env = str("APP_ENV", "development")
	switch c.Env {
	case "development", "production", "test":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, production or test, got %q", c.Env)
	}

	if c.Port, err = positiveInt("PORT", "3000"); err != nil {
		return Config{}, err
	}
	c.Debug = str("DEBUG", "false") == "true"

	tz := str("APP_TIMEZONE", "Europe/Moscow")
	if c.Timezone, err = time.LoadLocation(tz); err != nil {
		return Config{}, fmt.Errorf("APP_TIMEZONE is invalid: %w", err)
	}

	if c.DefaultPageSize, err = positiveInt("DEFAULT_PAGE_SIZE", "100"); err != nil {
		return Config{}, err
	}
	c.DatabaseURL = str("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/citypark_dev?sslmode=disable")

	c.JWT.Secret = str("JWT_SECRET", defaultJWTSecret)
	expires := str("JWT_EXPIRES_IN", "24h")
	if c.JWT.ExpiresIn, err = time.ParseDuration(expires); err != nil {
		return Config{}, fmt.Errorf("JWT_EXPIRES_IN is invalid: %w", err)
	}
	if c.Env == "production" && c.JWT.Secret == defaultJWTSecret {
		return Config{}, errors.New("JWT_SECRET must be changed in production")
	}

	c.Admin = Admin{
		Email:    str("ADMIN_EMAIL", "dev@dev.io"),
		Password: str("ADMIN_PASSWORD", "devdevdev"),
		Phone:    str("ADMIN_PHONE", "+79990000000"),
	}

	c.Iiko = Iiko{
		BaseURL:        str("IIKO_BASE_URL", "https://api-ru.iiko.services/api"),
		APIKey:         str("IIKO_API_KEY", ""),
		AppID:          str("IIKO_APP_ID", ""),
		ClientSecret:   str("IIKO_CLIENT_SECRET", ""),
		OrganizationID: str("IIKO_ORGANIZATION_ID", ""),
		ExternalMenuID: str("IIKO_EXTERNAL_MENU_ID", ""),
		UpdateToken:    str("IIKO_UPDATE_TOKEN", ""),
	}

	c.Telegram = Telegram{
		BaseURL:              str("TELEGRAM_BASE_URL", "https://api.telegram.org"),
		Token:                str("TELEGRAM_TOKEN", ""),
		ChatIDOrderCreated:   str("TELEGRAM_CHAT_ID_FOR_ORDER_CREATED", ""),
		ChatIDOrderConfirmed: str("TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED", ""),
	}
	c.Moneta = Moneta{
		ID:       str("MONETA_ID", ""),
		Secret:   str("MONETA_SECRET", ""),
		TestMode: str("MONETA_TEST_MODE", "false") == "true",
	}
	c.WorkingHours = WorkingHours{
		Start: str("WORKING_HOURS_START", "9:00"),
		End:   str("WORKING_HOURS_END", "22:00"),
	}
	c.ThankYouPageCash = str("THANK_YOU_PAGE_PAYMENT_CASH", "http://localhost:3001/thankyou")

	return c, nil
}

func (c Config) IsProduction() bool { return c.Env == "production" }
func (c Config) IsTest() bool       { return c.Env == "test" }
