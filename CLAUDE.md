# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Layout

This folder is not a repository itself. It holds two independent projects for «Сити Парк» (hotel, restaurant and
food delivery in Белореченск). Each one has its own git repository and its own `CLAUDE.md`, which hold the real rules.
Read the project's `CLAUDE.md` before you change anything in it, and run git commands inside the project folder.

| Folder | What it is | Stack | Detailed rules |
|---|---|---|---|
| `citypark-go/` | Backend for food ordering: menu synced from the iiko POS, payments through MONETA.RU, staff notifications through Telegram | Go, PostgreSQL 17 (Docker), embedded migrations | `citypark-go/CLAUDE.md`, `docs/tasks/CONVENTIONS.md` |
| `citypark-modern/` | Public website (in Russian), built from a finished design | Next.js 16 App Router, React 19, TypeScript, Tailwind v4, Feature-Sliced Design | `citypark-modern/CLAUDE.md`, `AGENTS.md`, `docs/PLAN.md` |
| `citypark-admin/` | Admin panel (in Russian): delivery prices, orders, menu photos and sync, site content, hotel reports | Vite SPA, React 19, HeroUI v3, TanStack Router + Query, types from the backend's OpenAPI | `citypark-admin/CLAUDE.md`, `docs/tasks/CONVENTIONS.md` |

Cross-project decisions are in `docs/adr/` (start with `0001-admin-panel.md`) and terms in `GLOSSARY.md`.

## Commands

Backend (`cd citypark-go`):

```bash
cp .env.example .env              # once
make db-up && make seed && make run   # dev DB on :5432, server on :3000
make test                         # starts the test DB on :5433, runs go test -p 1 ./...
make lint                         # gofmt -l . && go vet ./...
# single test (agents: never start the DB; use TEST_DB_DSN from .env.development, ask the user if it is down)
set -a; source .env.development; set +a; go test -p 1 -run TestName ./internal/order/
```

Tests in different packages share one database, so always keep `-p 1`.

Frontend (`cd citypark-modern`):

```bash
npm run dev                       # http://localhost:3000
npm run check                     # typecheck + eslint + Steiger (FSD layer rules); must be clean
npm run build && npm start        # needed before the qa:* scripts
npm run qa:flows                  # Playwright user flows; also qa:a11y, qa:responsive, qa:keyboard
node scripts/qa/flows/menu.mjs    # a single flow
```

The QA scripts use `BASE_URL` (default `http://localhost:3111`) and need `npx playwright install chromium` once.

## How the two projects relate

- They are **not connected yet**. The website has no backend: forms are Server Actions that send through
  `citypark-modern/src/shared/api/leads.ts`, which only logs. Menu, rooms and articles are typed data files read through
  async getters in each entity's `api/` segment. This is the seam where calls to `citypark-go` will go later
  (see "Backend: seams now, integrations later" in `citypark-modern/docs/PLAN.md`).
- The backend's API contract for the frontend is `citypark-go/docs/API.md` (OpenAPI in
  `citypark-go/internal/apidocs/openapi.yaml`). A change to an endpoint must update `docs/API.md`.
- Both dev servers use port **3000** by default. To run them together, change one of them (the backend expects the
  frontend at `:3001`, see `THANK_YOU_PAGE_PAYMENT_CASH`).

## Project-specific rules worth knowing up front

- `citypark-go` is built through numbered tasks: take the first unchecked task in `docs/tasks/PROGRESS.md`, do only
  that task, tick it and make one commit `task NN: <title>`. If a task cannot be done as written, stop and write the
  reason in `PROGRESS.md`. Never open `internal/iiko/testdata/` (large JSON fixtures).
- `citypark-modern` uses a Next.js version newer than your training data: read the guide in
  `node_modules/next/dist/docs/` before writing Next.js code. Colours come only from tokens
  (`design/tokens.css` is the source of truth); copy texts and prices exactly from `design/reference/`.
- `citypark-admin` uses the same numbered-task workflow. It uses HeroUI **v3**, which differs a lot from v2: read its
  docs before using a component. Many of its tasks need a citypark-go task first (noted in its `PROGRESS.md`). The admin's
  dev server runs on `:5173` (Vite default) and expects the backend at `VITE_API_URL`.

## Agent skills

### Issue tracker

Issues live in GitHub Issues for `drewp1ter/citypark` (via the `gh` CLI). See `docs/agents/issue-tracker.md`.

### Triage labels

Default vocabulary: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
