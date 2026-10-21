// Command seed fills an empty development database with sample data.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/drewp1ter/citypark-go/internal/config"
	"github.com/drewp1ter/citypark-go/internal/seed"
	"github.com/drewp1ter/citypark-go/internal/store"
)

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
	if cfg.IsProduction() {
		return errors.New("seed: refusing to run in production")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := store.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	sum, err := seed.Run(ctx, db)
	if errors.Is(err, seed.ErrNotEmpty) {
		fmt.Println("seed: database is not empty, nothing done")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Printf("seeded: %d users, %d categories, %d products, %d delivery prices, %d orders\n",
		sum.Users, sum.Categories, sum.Products, sum.DeliveryPrices, sum.Orders)
	return nil
}
