package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/drewp1ter/citypark-go/internal/app"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/config"
	"github.com/drewp1ter/citypark-go/internal/logging"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/user"
)

const shutdownTimeout = 20 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}

	log := logging.New(os.Stdout, cfg.Env, cfg.Debug)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close() // runs after the notifier has drained, see below

	if err := store.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	created, err := user.EnsureAdmin(ctx, user.NewStore(db), user.AdminParams{
		Email:    cfg.Admin.Email,
		Password: cfg.Admin.Password,
		Phone:    cfg.Admin.Phone,
	})
	if err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}
	if created {
		log.Info("admin account created", "email", cfg.Admin.Email)
	}

	a := app.New(cfg, db, log, clock.System(cfg.Timezone))
	srv := &http.Server{
		Addr:              "0.0.0.0:" + strconv.Itoa(cfg.Port),
		Handler:           a.Handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", srv.Addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("server stopped: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown: %w", err)
	}
	a.Notifier.Wait()
	return nil
}
