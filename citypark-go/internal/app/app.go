package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/apidocs"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/config"
	"github.com/drewp1ter/citypark-go/internal/content"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/deliveryprice"
	"github.com/drewp1ter/citypark-go/internal/deliverysettings"
	"github.com/drewp1ter/citypark-go/internal/google"
	"github.com/drewp1ter/citypark-go/internal/health"
	"github.com/drewp1ter/citypark-go/internal/homepagemenu"
	"github.com/drewp1ter/citypark-go/internal/hotel"
	"github.com/drewp1ter/citypark-go/internal/iiko"
	"github.com/drewp1ter/citypark-go/internal/live"
	"github.com/drewp1ter/citypark-go/internal/logging"
	"github.com/drewp1ter/citypark-go/internal/mailer"
	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/news"
	"github.com/drewp1ter/citypark-go/internal/onec"
	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/promo"
	"github.com/drewp1ter/citypark-go/internal/room"
	"github.com/drewp1ter/citypark-go/internal/settings"
	"github.com/drewp1ter/citypark-go/internal/telegram"
	"github.com/drewp1ter/citypark-go/internal/upload"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/drewp1ter/citypark-go/internal/workinghours"
)

// App is the assembled application. Notifier and IikoOrders are exposed so the caller can wait for pending sends.
type App struct {
	Handler    http.Handler
	Notifier   *TelegramNotifier
	IikoOrders *IikoOrders
	Hub        *live.Hub
	Syncer     *iiko.Syncer
	Uploads    *upload.Storage
}

// New builds all stores, services and handlers and returns the router.
func New(cfg config.Config, db *sqlx.DB, log *slog.Logger, c clock.Clock, middleware ...func(http.Handler) http.Handler) *App {
	if cfg.Moneta.ID == "" || cfg.Moneta.Secret == "" {
		log.Warn("moneta is not configured")
	}

	tokens := auth.NewTokens(cfg.JWT.Secret, cfg.JWT.ExpiresIn, c)
	sessions := customer.NewService(db, c)
	if cfg.SMTP.Enabled {
		sessions.ConfigureEmail(mailer.NewSMTP(cfg.SMTP, nil), cfg.CustomerPublicOrigin, nil, log)
	}
	adapter := &customer.Adapter{Sessions: sessions, Tokens: tokens, Enabled: cfg.CustomerAuthEnabled}

	telegramClient := telegram.NewClient(cfg.Telegram.BaseURL, cfg.Telegram.Token, nil)
	notifier := NewTelegramNotifier(telegramClient, cfg.Telegram.StaffChatID, log)
	payments := NewMonetaPayments(
		moneta.New(cfg.Moneta.ID, cfg.Moneta.Secret, cfg.Moneta.TestMode),
		PaymentPages{Base: cfg.Moneta.URL, Success: cfg.Moneta.SuccessURL, Fail: cfg.Moneta.FailURL},
	)

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

	iikoOrders := NewIikoOrders(iikoClient, db, telegramClient, cfg.Telegram.StaffChatID, IikoOrderSettings{
		PromoDiscountTypeID: cfg.Iiko.PromoDiscountTypeID,
		TerminalGroupID:     cfg.Iiko.TerminalGroupID,
		PaymentTypeID:       cfg.Iiko.PaymentTypeID,
		PaymentTypeKind:     cfg.Iiko.PaymentTypeKind,
		City:                cfg.Iiko.AddressCity,
		Street:              cfg.Iiko.AddressStreet,
		House:               cfg.Iiko.AddressHouse,
		Delays:              []time.Duration{30 * time.Second, 2 * time.Minute},
	}, log)

	hub := live.NewHub(log)
	orders := order.NewService(db, c, Notifiers{notifier, iikoOrders, hub}, payments)
	if cfg.Iiko.TerminalGroupID != "" && cfg.Iiko.PaymentTypeID != "" {
		orders.ConfigurePromo(cfg.Iiko.PromoDiscountTypeID)
	}
	hours := workinghours.NewStore(db)
	orders.SetPaymentHours(hours)
	googleService := google.NewService(db, c, cfg.Google.ClientID, cfg.Google.CallbackURL, "")
	googleService.ConfigureCompletion(cfg.Google.ClientSecret, "", nil)

	r := chi.NewRouter()
	r.Use(logging.Recover(log))
	r.Use(middleware...)
	r.Use(logging.Requests(log), cors)
	r.Route("/api/v1", func(r chi.Router) {
		health.Routes(r)
		apidocs.Routes(r)
		user.NewAuthHandler(users, tokens, cfg.CustomerAuthEnabled).Routes(r)
		user.NewHandler(users, tokens, adapter.Required(true)).Routes(r)
		customer.NewHandler(users, customer.NewStore(db), sessions, adapter).Routes(r)
		customer.NewPairingHandler(sessions, customer.NewLimits(c), cfg.CustomerAuthEnabled).Routes(r)
		customer.NewEmailHandler(db, sessions, customer.NewLimits(c), cfg.CustomerAuthEnabled).Routes(r)
		customer.NewTelegramHandler(db, sessions, customer.NewLimits(c), cfg.CustomerTelegramBotToken, cfg.CustomerAuthEnabled).Routes(r)
		google.NewHandler(googleService, sessions, customer.NewLimits(c), cfg.Google.Enabled && cfg.CustomerAuthEnabled).Routes(r)
		address.NewHandler(address.NewStore(db), tokens, adapter.Required(true)).Routes(r)
		deliveryprice.NewHandler(db, tokens).Routes(r)
		promo.NewHandler(db, tokens).Routes(r)
		catalogHandler := catalog.NewHandler(catalog.NewStore(db), c, cfg.DefaultPageSize)
		catalogHandler.Routes(r)
		catalogHandler.AdminRoutes(r, tokens)
		iiko.NewHandler(syncer, cfg.Iiko.UpdateToken, tokens).Routes(r)
		hotel.NewHandler(hotel.NewStore(db), cfg.Hotel.WebhookSecret, tokens).Routes(r)
		room.NewHandler(room.NewStore(db), uploads, tokens).Routes(r)
		news.NewHandler(db, uploads, tokens, c, cfg.DefaultPageSize).Routes(r)
		content.NewHandler(db, uploads, tokens).Routes(r)
		settings.NewHandler(settings.NewStore(db), tokens).Routes(r)
		workinghours.NewHandler(hours, tokens).Routes(r)
		deliverysettings.NewHandler(deliverysettings.NewStore(db), tokens).Routes(r)
		homepagemenu.NewHandler(homepagemenu.NewStore(db), tokens).Routes(r)
		live.NewHandler(hub, tokens, log).Routes(r)
		order.NewHandler(orders, tokens, order.HandlerOptions{
			Hours:           hours,
			DefaultPageSize: cfg.DefaultPageSize,
		}, adapter.Required(true), adapter.Optional).Routes(r)
		onec.NewHandler(onec.NewClient(cfg.OneC.HotelURL, cfg.OneC.User, cfg.OneC.Password, nil), tokens, c).Routes(r)
		upload.NewHandler(uploads, tokens).Routes(r)
	})
	// Static files live outside /api/v1.
	r.Get("/uploads/*", uploads.Serve().ServeHTTP)
	r.Head("/uploads/*", uploads.Serve().ServeHTTP)

	return &App{Handler: r, Notifier: notifier, IikoOrders: iikoOrders, Hub: hub, Syncer: syncer, Uploads: uploads}
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
