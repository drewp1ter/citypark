package store

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"github.com/drewp1ter/citypark-go/migrations"
)

// Migrate applies all pending migrations from the embedded files.
func Migrate(ctx context.Context, db *sqlx.DB) error {
	p, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
	if err != nil {
		return err
	}
	_, err = p.Up(ctx)
	return err
}

// MigrateDown rolls back every migration. Used by tests only.
func MigrateDown(ctx context.Context, db *sqlx.DB) error {
	p, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
	if err != nil {
		return err
	}
	_, err = p.DownTo(ctx, 0)
	return err
}
