# Task 20: iiko client

## Goal
An HTTP client for the two iiko Cloud API calls the service needs: get an access token, download the nomenclature (menu).

## Depends on
01.

## Read first
`docs/tasks/CONVENTIONS.md`.

## Files to create
- `internal/iiko/types.go`
- `internal/iiko/client.go`
- `internal/iiko/client_test.go`

Do not open `internal/iiko/testdata/*.json` (large fixtures, used in task 22).

## Spec

### `types.go`
Only the fields the service uses. Unknown JSON fields are ignored.
```go
package iiko

type Nomenclature struct {
    Groups   []Group   `json:"groups"`
    Products []Product `json:"products"`
}

type Group struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    Description    *string  `json:"description"`
    Order          int      `json:"order"`
    AdditionalInfo *string  `json:"additionalInfo"`
    ImageLinks     []string `json:"imageLinks"`
    IsDeleted      bool     `json:"isDeleted"`
}

type Product struct {
    ID                  string      `json:"id"`
    Name                string      `json:"name"`
    Description         *string     `json:"description"`
    ParentGroup         *string     `json:"parentGroup"`
    Order               int         `json:"order"`
    AdditionalInfo      *string     `json:"additionalInfo"`
    ImageLinks          []string    `json:"imageLinks"`
    Tags                []string    `json:"tags"`
    IsDeleted           bool        `json:"isDeleted"`
    Weight              float64     `json:"weight"`
    MeasureUnit         string      `json:"measureUnit"`
    FatAmount           float64     `json:"fatAmount"`
    ProteinsAmount      float64     `json:"proteinsAmount"`
    CarbohydratesAmount float64     `json:"carbohydratesAmount"`
    EnergyAmount        float64     `json:"energyAmount"`
    SizePrices          []SizePrice `json:"sizePrices"`
}

type SizePrice struct {
    Price struct {
        CurrentPrice float64 `json:"currentPrice"`
    } `json:"price"`
}
```

### `client.go`
```go
type Client struct { /* baseURL, apiLogin, organizationID string; http *http.Client; mu sync.Mutex; token string */ }

// NewClient: baseURL without a trailing slash, e.g. "https://api-ru.iiko.services/api/1".
// A nil httpClient means &http.Client{Timeout: 60 * time.Second}.
func NewClient(baseURL, apiLogin, organizationID string, httpClient *http.Client) *Client

var ErrUnauthorized = errors.New("iiko: unauthorized")

// AccessToken requests a new token.
func (c *Client) AccessToken(ctx context.Context) (string, error)

// Nomenclature downloads the menu using the given token.
func (c *Client) Nomenclature(ctx context.Context, token string) (*Nomenclature, error)

// Fetch returns the menu, managing the token itself.
func (c *Client) Fetch(ctx context.Context) (*Nomenclature, error)
```

| Call | Request | Response handling |
|---|---|---|
| `AccessToken` | `POST {baseURL}/access_token`, header `Content-Type: application/json`, body `{"apiLogin": "<apiLogin>"}` | 200: JSON `{"token": "..."}` → return token (empty token is an error). 401: `ErrUnauthorized`. Other status: error `iiko: access_token: unexpected status <code>`. |
| `Nomenclature` | `POST {baseURL}/nomenclature`, headers `Content-Type: application/json`, `Authorization: Bearer <token>`, body `{"organizationId": "<organizationID>", "startRevision": 0}` | 200: decode `Nomenclature`. 401: `ErrUnauthorized`. Other status: error `iiko: nomenclature: unexpected status <code>`. |

`Fetch`:
1. Lock `mu` for the whole call (one sync at a time).
2. If the cached token is empty, call `AccessToken` and cache it.
3. Call `Nomenclature`. On success return.
4. If the error is `ErrUnauthorized` (use `errors.Is`): clear the token, get a new one, call `Nomenclature` once more and return its result.
5. Any other error is returned at once.

Always close response bodies. Build requests with `http.NewRequestWithContext`.

## Tests to write
Use `httptest.NewServer` with a handler that counts requests per path.

- `TestAccessToken`: the server checks method, path, content type and the `apiLogin` in the body, and answers `{"token":"t1"}` → `"t1"`.
- `TestAccessTokenUnauthorized`: 401 → `errors.Is(err, ErrUnauthorized)`.
- `TestAccessTokenEmpty`: `{"token":""}` → error.
- `TestNomenclature`: the server checks the `Authorization` header and `organizationId` in the body and answers a small JSON with one group and one product (with `sizePrices[0].price.currentPrice = 320`, `description: null`, `tags: null`, and an unknown extra field) → fields decoded, `Description == nil`.
- `TestNomenclatureServerError`: 500 → error that is not `ErrUnauthorized`.
- `TestFetchRefreshesToken`: the server's first `/access_token` answer is `bad_token`, later answers are `good_token`; `/nomenclature` answers 401 for `Bearer bad_token` and 200 for `Bearer good_token`. `Fetch` succeeds and the server saw exactly 4 requests. A second `Fetch` adds exactly 1 request (the token is cached).
- `TestFetchGivesUp`: `/nomenclature` always answers 401 → `Fetch` returns `ErrUnauthorized` after 2 nomenclature requests.
- `TestFetchContextCancelled`: a cancelled context returns an error.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Saving anything to the database (tasks 21, 22).
