// Package config reads application settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                  string // "development", "production" or "test"
	Port                 int
	Debug                bool
	Timezone             *time.Location
	DefaultPageSize      int
	DatabaseURL          string
	JWT                  JWT
	CustomerAuthEnabled  bool
	CustomerPublicOrigin string
	SMTP                 SMTP
	Admin                Admin
	Iiko                 Iiko
	Telegram             Telegram
	Moneta               Moneta
	Hotel                Hotel
	OneC                 OneC
	Upload               Upload
}

type JWT struct {
	Secret    string
	ExpiresIn time.Duration
}
type Admin struct{ Email, Password, Phone string }
type Iiko struct {
	BaseURL, APIKey, AppID, ClientSecret, OrganizationID, ExternalMenuID, UpdateToken string
	SyncInterval                                                                      time.Duration // 0 = no scheduled sync

	// Orders sent to iiko. Without TerminalGroupID and PaymentTypeID orders are not sent.
	TerminalGroupID, PaymentTypeID, PaymentTypeKind string
	// Placeholder address of the iiko order: iiko's legacy address needs a street and a house of its own.
	AddressCity, AddressStreet, AddressHouse string
}
type Telegram struct{ BaseURL, Token, ChatIDOrderCreated, ChatIDOrderConfirmed string }
type Moneta struct {
	ID         string // shop id, sent to Moneta as MNT_ID
	Secret     string // integrity code from the Moneta cabinet
	TestMode   bool
	URL        string // MONETA.Assistant address the customer is sent to
	SuccessURL string // MNT_SUCCESS_URL; empty = not sent
	FailURL    string // MNT_FAIL_URL; empty = not sent
}
type Hotel struct{ WebhookSecret string }
type OneC struct{ HotelURL, User, Password string }
type Upload struct{ Dir, PublicBaseURL string }
type SMTP struct {
	Host                              string
	Port                              int
	Username, Password, From, TLSMode string
	Enabled                           bool
}

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
	c.CustomerAuthEnabled = str("CUSTOMER_AUTH_ENABLED", "false") == "true"
	c.CustomerPublicOrigin = str("CUSTOMER_PUBLIC_ORIGIN", "http://localhost:3001")
	if err := validateCustomerOrigin(c.CustomerPublicOrigin, c.Env); err != nil {
		return Config{}, err
	}
	c.SMTP = SMTP{Host: str("SMTP_HOST", ""), Username: str("SMTP_USERNAME", ""), Password: str("SMTP_PASSWORD", ""), From: str("SMTP_FROM", ""), TLSMode: str("SMTP_TLS_MODE", "starttls")}
	if c.SMTP.Port, err = positiveInt("SMTP_PORT", "587"); err != nil || c.SMTP.Port > 65535 {
		return Config{}, errors.New("SMTP_PORT must be between 1 and 65535")
	}
	if c.SMTP.TLSMode != "starttls" && c.SMTP.TLSMode != "tls" {
		return Config{}, errors.New("SMTP_TLS_MODE must be starttls or tls")
	}
	c.SMTP.Enabled = c.SMTP.Host != "" && c.SMTP.Username != "" && c.SMTP.Password != "" && c.SMTP.From != ""
	if c.CustomerAuthEnabled {
		if c.SMTP.Host == "" || c.SMTP.Username == "" || c.SMTP.Password == "" || c.SMTP.From == "" {
			return Config{}, errors.New("SMTP configuration is incomplete")
		}
	}
	if c.SMTP.Host != "" || c.SMTP.Username != "" || c.SMTP.From != "" {
		if strings.ContainsAny(c.SMTP.Host+c.SMTP.From, "\r\n") || strings.ContainsAny(c.SMTP.Username, "\r\n") {
			return Config{}, errors.New("SMTP headers contain a newline")
		}
		if c.SMTP.From != "" {
			address, e := mail.ParseAddress(c.SMTP.From)
			if e != nil || address.Address != c.SMTP.From {
				return Config{}, errors.New("SMTP_FROM must be a single email address")
			}
		}
	}

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

		TerminalGroupID: str("IIKO_TERMINAL_GROUP_ID", ""),
		PaymentTypeID:   str("IIKO_PAYMENT_TYPE_ID", ""),
		PaymentTypeKind: str("IIKO_PAYMENT_TYPE_KIND", "Card"),
		AddressCity:     str("IIKO_ADDRESS_CITY", "Белореченск"),
		AddressStreet:   str("IIKO_ADDRESS_STREET", "Гоголя"),
		AddressHouse:    str("IIKO_ADDRESS_HOUSE", "61"),
	}
	syncInterval := str("IIKO_SYNC_INTERVAL", "0")
	if c.Iiko.SyncInterval, err = time.ParseDuration(syncInterval); err != nil {
		return Config{}, fmt.Errorf("IIKO_SYNC_INTERVAL is invalid: %w", err)
	}
	if c.Iiko.SyncInterval < 0 {
		return Config{}, fmt.Errorf("IIKO_SYNC_INTERVAL is invalid: must not be negative, got %q", syncInterval)
	}

	c.Telegram = Telegram{
		BaseURL:              str("TELEGRAM_BASE_URL", "https://api.telegram.org"),
		Token:                str("TELEGRAM_TOKEN", ""),
		ChatIDOrderCreated:   str("TELEGRAM_CHAT_ID_FOR_ORDER_CREATED", ""),
		ChatIDOrderConfirmed: str("TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED", ""),
	}
	c.Moneta = Moneta{
		ID:         str("MONETA_ID", ""),
		Secret:     str("MONETA_SECRET", ""),
		TestMode:   str("MONETA_TEST_MODE", "false") == "true",
		URL:        str("MONETA_URL", "https://assistant.demo.moneta.ru/"),
		SuccessURL: str("MONETA_SUCCESS_URL", "http://localhost:3001/checkout/done"),
		FailURL:    str("MONETA_FAIL_URL", "http://localhost:3001/checkout?payment=failed"),
	}
	c.Hotel = Hotel{WebhookSecret: str("HOTEL_WEBHOOK_SECRET", "")}
	c.OneC = OneC{
		HotelURL: str("ONEC_HOTEL_URL", ""),
		User:     str("ONEC_USER", ""),
		Password: str("ONEC_PASSWORD", ""),
	}
	c.Upload = Upload{
		Dir:           str("UPLOAD_DIR", "./uploads"),
		PublicBaseURL: strings.TrimRight(str("PUBLIC_BASE_URL", "http://localhost:3000"), "/"),
	}

	return c, nil
}

func (c Config) IsProduction() bool { return c.Env == "production" }
func (c Config) IsTest() bool       { return c.Env == "test" }

func validateCustomerOrigin(raw, env string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return errors.New("CUSTOMER_PUBLIC_ORIGIN must be an absolute origin")
	}
	if u.Scheme == "https" {
		return nil
	}
	if env != "production" && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
		return nil
	}
	return errors.New("CUSTOMER_PUBLIC_ORIGIN must use HTTPS")
}
