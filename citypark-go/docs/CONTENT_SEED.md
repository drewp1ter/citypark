# Production content seed

The committed `seeds/production` bundle contains the public content exported from the current database on 2026-11-03: 76 news articles, 58 delivery tariffs, 101 content blocks, 5 room types, and 88 referenced images (77 uploads). It contains no customer or order data and no database credentials.

Included tables: news, delivery_prices, content_blocks, rooms, site_settings, delivery_settings, working_hours, hotel_prices, breakfast_items. Excluded: customer accounts and addresses, orders, sessions, payments, promotions, live room availability, iiko catalogs, and homepage_menu (currently empty and catalog-dependent).

The existing demo seeder is unchanged. No HTTP interfaces or historical migrations change.

## Commands

Set DATABASE_URL explicitly to the intended database. The command does not load .env automatically. Apply existing migrations first using the normal server startup, then stop application writes for restore. Back up the target database and uploads before restoring.

```bash
# Export uses a read-only, repeatable-read transaction and requires a new directory.
go run ./cmd/content-seed export --bundle /tmp/content-export --uploads ./uploads --images ../citypark-modern/public/images

# Check the committed bundle against the target schema and shipped website images.
go run ./cmd/content-seed check --bundle seeds/production --images ../citypark-modern/public/images

# Explicitly replace ONLY the included datasets with the snapshot.
go run ./cmd/content-seed restore --bundle seeds/production --uploads ./uploads --images ../citypark-modern/public/images --replace
```

Restore deletes target-only rows in included tables, preserves source IDs and timestamps, and restarts identity sequences transactionally. Repeating restore produces the same content. It intentionally overwrites subsequent editorial changes; never run automatically on startup.

All referenced upload files ship in the bundle. `/images/` files ship with the website and must be available for checksum validation. Rich-text references are included. Missing, altered, or conflicting assets cause failure. Uploads are copied before the database transaction commits; failed restores may leave harmless additional files, but unrelated files are never deleted. Database failures roll back all selected datasets and sequence changes.

The Docker image includes `/app/content-seed` and `/app/seeds/production`. Invoke the binary as a one-off command with DATABASE_URL, the persistent upload volume, and a read-only mount of the website images, e.g. `--images /site-images --bundle /app/seeds/production --uploads /data/uploads`. Override the default server entrypoint to run the tool. The tool needs no PostgreSQL CLI utilities.

To refresh the committed seed, export into a new directory, review snapshot changes and image checksums, validate it, then replace the bundle in a reviewed commit. Export never modifies source rows. Stop editorial changes during snapshot packaging if uploaded files might be removed concurrently.
