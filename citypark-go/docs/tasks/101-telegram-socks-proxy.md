# Task 101: Telegram-only SOCKS proxy

## Goal
Route outbound Telegram Bot API requests through an optional SOCKS5 proxy without changing other clients.

## Depends on
100

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/config/config.go`, `internal/telegram/client.go`, `internal/app/app.go`.

## Files to create
- `internal/telegram/proxy_test.go`

## Files to change
- `internal/config/config.go` and `internal/config/config_test.go`: parse and validate optional proxy URL.
- `internal/telegram/client.go`: isolated HTTP transport constructor.
- `internal/app/app.go`: wire the Telegram transport.
- `.env.example` and `README.md`: document configuration.
- `docs/tasks/PROGRESS.md`: record completion.

## Spec
Add `Telegram.ProxyURL *url.URL`, loaded from `TELEGRAM_PROXY_URL`. Accept socks5/socks5h with hostname and port 1–65535, optional credentials, no path/query/fragment. Reject invalid values without printing credentials. Export `telegram.NewProxyHTTPClient(proxyURL *url.URL) *http.Client`; clone the default transport, use the explicit proxy when configured, retain the 15-second timeout. An unavailable proxy must fail without direct fallback. Empty preserves current behavior. Other clients remain unchanged.

## Tests to write
- `TestTelegramProxyConfig`: accepted schemes, credentials, empty configuration and invalid values.
- `TestProxyHTTPClient`: fake SOCKS server observes target hostname, credentials and Bot API request; proxy failure has no direct fallback; global transport is unchanged.

## Done when
All CONVENTIONS checks pass; changed internal packages have at least 80% coverage.

## Out of scope
Live proxy setup, deployment, real notifications and API endpoint changes.
