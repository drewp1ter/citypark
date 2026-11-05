# City Park (Сити Парк)

Monorepo for the «Сити Парк» hotel, restaurant and delivery in Белореченск.

| Folder | What it is |
| --- | --- |
| `citypark-go/` | Ordering backend (Go, PostgreSQL, iiko, MONETA.RU, Telegram) |
| `citypark-modern/` | Website (Next.js, Tailwind, Feature-Sliced Design) |

Each project has its own README and `CLAUDE.md`. From the root, `make go-test`, `make go-lint`, `make web-check`
and so on forward to the projects (see `Makefile`).

## Production deployment

See [deploy/README.md](deploy/README.md) for building AMD64 image archives locally and deploying them without a source checkout,
including runtime configuration, existing host nginx integration, and backup/restore commands.
