package app

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/apidocs"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/config"
	"github.com/drewp1ter/citypark-go/internal/content"
	"github.com/drewp1ter/citypark-go/internal/deliveryprice"
	"github.com/drewp1ter/citypark-go/internal/health"
	"github.com/drewp1ter/citypark-go/internal/hotel"
	"github.com/drewp1ter/citypark-go/internal/iiko"
	"github.com/drewp1ter/citypark-go/internal/logging"
	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/news"
	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/room"
	"github.com/drewp1ter/citypark-go/internal/settings"
	"github.com/drewp1ter/citypark-go/internal/telegram"
	"github.com/drewp1ter/citypark-go/internal/upload"
	"github.com/drewp1ter/citypark-go/internal/user"
)

// App is the assembled application. Notifier is exposed so the caller can wait for pending sends.
type App struct {
	Handler  http.Handler
	Notifier *TelegramNotifier
	Syncer   *iiko.Syncer
	Uploads  *upload.Storage
}

// New builds all stores, services and handlers and returns the router.
func New(cfg config.Config, db *sqlx.DB, log *slog.Logger, c clock.Clock) *App {
	if cfg.Moneta.ID == "" || cfg.Moneta.Secret == "" {
		log.Warn("moneta is not configured")
	}

	tokens := auth.NewTokens(cfg.JWT.Secret, cfg.JWT.ExpiresIn, c)

	telegramClient := telegram.NewClient(cfg.Telegram.BaseURL, cfg.Telegram.Token, nil)
	notifier := NewTelegramNotifier(telegramClient, cfg.Telegram.ChatIDOrderCreated, cfg.Telegram.ChatIDOrderConfirmed, log)
	payments := NewMonetaPayments(moneta.New(cfg.Moneta.ID, cfg.Moneta.Secret, cfg.Moneta.TestMode))

	users := user.NewStore(db)
	iikoClient := iiko.NewClient(
		cfg.Iiko.BaseURL,
		iiko.Credentials{APIKey: cfg.Iiko.APIKey, AppID: cfg.Iiko.AppID, ClientSecret: cfg.Iiko.ClientSecret},
		cfg.Iiko.OrganizationID,
		cfg.Iiko.ExternalMenuID,
		nil,
	)
	syncer := iiko.NewSyncer(db, iikoClient)

	uploads := upload.NewStorage(cfg.Upload.Dir, cfg.Upload.PublicBaseURL)

	orders := order.NewService(db, c, notifier, payments, order.Options{ThankYouCash: cfg.ThankYouPageCash})

	r := chi.NewRouter()
	r.Use(logging.Recover(log), logging.Requests(log), cors)
	r.Route("/api/v1", func(r chi.Router) {
		health.Routes(r)
		apidocs.Routes(r)
		user.NewAuthHandler(users, tokens).Routes(r)
		user.NewHandler(users, tokens).Routes(r)
		address.NewHandler(address.NewStore(db), tokens).Routes(r)
		deliveryprice.NewHandler(db, tokens).Routes(r)
		catalog.NewHandler(catalog.NewStore(db), c, cfg.DefaultPageSize).Routes(r)
		iiko.NewHandler(syncer, cfg.Iiko.UpdateToken, tokens).Routes(r)
		hotel.NewHandler(hotel.NewStore(db), cfg.Hotel.WebhookSecret, tokens).Routes(r)
		room.NewHandler(room.NewStore(db), uploads, tokens).Routes(r)
		news.NewHandler(db, uploads, tokens, c, cfg.DefaultPageSize).Routes(r)
		content.NewHandler(db, uploads, tokens).Routes(r)
		settings.NewHandler(settings.NewStore(db), tokens).Routes(r)
		order.NewHandler(orders, tokens, order.HandlerOptions{
			WorkingHoursStart: cfg.WorkingHours.Start,
			WorkingHoursEnd:   cfg.WorkingHours.End,
			DefaultPageSize:   cfg.DefaultPageSize,
		}).Routes(r)
		upload.NewHandler(uploads, tokens).Routes(r)
	})
	// Static files live outside /api/v1.
	r.Get("/uploads/*", uploads.Serve().ServeHTTP)
	r.Head("/uploads/*", uploads.Serve().ServeHTTP)

	return &App{Handler: r, Notifier: notifier, Syncer: syncer, Uploads: uploads}
}

// cors allows any origin and answers preflight requests itself.
// It runs before routing, so OPTIONS needs no route of its own.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Update-Token, X-Webhook-Secret")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
