# Task 36: iiko v2 auth and config

## Goal
The iiko client gets its token from the new `POST /api/v2/access_token` endpoint, using `apiKey`, `appId` and `clientSecret` instead of `apiLogin`. Every authorized call refreshes the token once on a 401. `IIKO_API_LOGIN` and `IIKO_CATEGORY_MARKER` are gone; the new credentials and the menu id come from new env vars.

## Depends on
33.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/client.go`, `internal/config/config.go`.

Never open `internal/iiko/testdata/*.json`.

## Files to change
- `internal/config/config.go`: the new `Iiko` struct and env vars below; remove the empty-marker check.
- `internal/config/config_test.go`: follow the new fields; remove the "empty marker" case.
- `internal/iiko/client.go`: new credentials, new access token request, `withToken`.
- `internal/iiko/client_test.go`: follow the new constructor; the tests below.
- `internal/app/app.go`: build the client from the new config. Until task 38, pass the literal `"#"` as the marker to `iiko.NewSyncer`.
- `.env.example`, `README.md`: the env table and example follow the config.

## Spec

### Config
```go
type Iiko struct{ BaseURL, APIKey, AppID, ClientSecret, OrganizationID, ExternalMenuID, UpdateToken string }
```

| Env | Field | Default |
|---|---|---|
| `IIKO_BASE_URL` | `Iiko.BaseURL` | `https://api-ru.iiko.services/api` |
| `IIKO_API_KEY` | `Iiko.APIKey` | empty |
| `IIKO_APP_ID` | `Iiko.AppID` | empty |
| `IIKO_CLIENT_SECRET` | `Iiko.ClientSecret` | empty |
| `IIKO_ORGANIZATION_ID` | `Iiko.OrganizationID` | empty |
| `IIKO_EXTERNAL_MENU_ID` | `Iiko.ExternalMenuID` | empty |
| `IIKO_UPDATE_TOKEN` | `Iiko.UpdateToken` | empty |

`IIKO_API_LOGIN` and `IIKO_CATEGORY_MARKER` are removed from the code, `.env.example` and `README.md`. Note that the base URL no longer ends in `/1`: the endpoints carry their own version.

`.env.example` gets empty values for the new secrets. Never put real credentials in any file.

### Client
```go
// Credentials authenticate the application in iiko.
type Credentials struct{ APIKey, AppID, ClientSecret string }

// NewClient: baseURL has no trailing slash, e.g. "https://api-ru.iiko.services/api".
// A nil httpClient means a client with a 60 second timeout.
func NewClient(baseURL string, creds Credentials, organizationID, externalMenuID string, httpClient *http.Client) *Client
```
`externalMenuID` is stored in the client for task 37 and is not used yet.

| Call | Request | Response handling |
|---|---|---|
| `AccessToken` | `POST {baseURL}/v2/access_token`, `Content-Type: application/json`, **no** `Authorization` header, body `{"apiKey": "...", "appId": "...", "clientSecret": "..."}` | 200: JSON `{"correlationId": "...", "token": "..."}` → return `token` (an empty token is an error). 401: `ErrUnauthorized`. Other: `iiko: v2/access_token: unexpected status <code>`. |
| `Nomenclature` | `POST {baseURL}/1/nomenclature`, `Authorization: Bearer <token>`; body unchanged | unchanged |

Every request made with a token sends the header `Authorization: Bearer <token>`. `post()` already does this when `token != ""`.

### Token refresh
Move the token logic out of `Fetch` into one private method that every authorized call uses:
```go
// withToken runs call with the cached token, getting one first if needed.
// If call returns ErrUnauthorized, it gets a new token and runs call once more.
// It holds c.mu for the whole call, so one iiko call runs at a time.
func (c *Client) withToken(ctx context.Context, call func(token string) error) error
```
1. Lock `mu`.
2. If the cached token is empty, call `AccessToken` and cache the result. On error return it.
3. Run `call(token)`. If the error is nil or not `ErrUnauthorized` (`errors.Is`), return it.
4. Clear the cached token, get a new one (on error return it), cache it, and return `call(newToken)`.

`Fetch` becomes:
```go
func (c *Client) Fetch(ctx context.Context) (*Nomenclature, error) {
    var n *Nomenclature
    err := c.withToken(ctx, func(token string) error {
        var err error
        n, err = c.Nomenclature(ctx, token)
        return err
    })
    return n, err
}
```

## Tests to write
Update `newTestClient` to `NewClient(srv.URL, Credentials{APIKey: "key-1", AppID: "app-1", ClientSecret: "secret-1"}, "org-1", "menu-1", nil)`. Change the paths in the existing tests to `/v2/access_token` and `/1/nomenclature`. Keep every existing test.

- `TestAccessToken`: the server checks the method, the path `/v2/access_token`, the content type, that the `Authorization` header is empty, and the body `apiKey=key-1`, `appId=app-1`, `clientSecret=secret-1`. It answers `{"correlationId":"c1","token":"t1"}` → `"t1"`.
- `TestNomenclature`: also checks `Authorization` == `Bearer t1` when called with token `t1`.
- `TestFetchSendsBearerToken`: `/v2/access_token` answers `{"token":"tok-9"}`; `/1/nomenclature` answers 401 unless `Authorization` is exactly `Bearer tok-9` → `Fetch` succeeds after 1 token request and 1 nomenclature request.
- `TestFetchRefreshesToken`, `TestFetchGivesUp`, `TestFetchAccessTokenError`, `TestFetchRefreshTokenError`, `TestFetchNomenclatureErrorIsNotRetried`: unchanged behaviour, new paths. Together they cover each branch of `withToken`.
- `TestFetchCachesToken`: two `Fetch` calls in a row → the server saw 1 token request and 2 nomenclature requests.
- Config: `TestLoadDefaults` checks `Iiko.BaseURL == "https://api-ru.iiko.services/api"` and that the other `Iiko` fields are empty. `TestLoadFromEnv` sets each new variable and checks its field. Remove the `IIKO_API_LOGIN` and `IIKO_CATEGORY_MARKER` entries and the "empty marker" error case.

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/iiko` and `internal/config` is at least 80%, and `grep -rn "IIKO_API_LOGIN\|IIKO_CATEGORY_MARKER\|APILogin" internal cmd .env.example README.md` prints nothing.

## Out of scope
The menu/v3 endpoint and the `menus` table (task 37). Changing the sync or removing `Nomenclature` (task 38). Do not edit old task files or `docs/design/`.
