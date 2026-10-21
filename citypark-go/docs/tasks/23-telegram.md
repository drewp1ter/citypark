# Task 23: Telegram client and order message

## Goal
Send messages through the Telegram Bot API and format the order text that staff receive.

## Depends on
09.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/money/money.go`.

## Files to create
- `internal/telegram/client.go`
- `internal/telegram/message.go`
- `internal/telegram/telegram_test.go`

This package must not import `internal/order` (the order package does not exist yet; task 31 connects them).

## Spec

### `client.go`
```go
package telegram

type Client struct { /* baseURL, token string; http *http.Client */ }

// NewClient: baseURL e.g. "https://api.telegram.org". nil httpClient means a 15 second timeout.
func NewClient(baseURL, token string, httpClient *http.Client) *Client

// SendMessage posts a Markdown message to a chat.
func (c *Client) SendMessage(ctx context.Context, chatID, text string) error
```
Request: `POST {baseURL}/bot{token}/sendMessage`, headers `Content-Type: application/json` and `Accept: application/json`, body:
```json
{ "chat_id": "<chatID>", "parse_mode": "Markdown", "text": "<text>" }
```
Response:
- status >= 500 → error `telegram: server error, status <code>`
- status >= 400 → decode the body `{"description": "..."}` and return error `telegram: <description>` (if the body cannot be decoded: `telegram: status <code>`)
- otherwise nil

Never put the token into an error message or a log line.

### `message.go`
```go
// OrderMessage is everything the text needs. It is filled by the caller.
type OrderMessage struct {
    ID          int64
    Name        string        // customer name
    Phone       string
    Address     string        // "city, street"; "" means pickup
    PaymentType string        // "cash" or "online"
    Total       money.Kopecks // sum of items, without delivery
    Delivery    money.Kopecks
    Comment     string
    Items       []OrderItem
}

type OrderItem struct {
    Category string
    Name     string
    Amount   int
}

func FormatOrder(m OrderMessage) string
```

Payment type text: `cash` → `наличными`, `online` → `онлайн`, anything else → the value itself.
Money is printed with `Kopecks.String()`.

With an address (delivery), the exact output is (labels are padded with spaces to 16 characters):
````
```
Номер заказа:   15
Имя:            Иван
Телефон:        +79991234567
Доставка:       Да
Адрес доставки: Сочи, ул. Ленина, 1
Оплата:         наличными
Сумма заказа:   640
Сумма доставки: 150
Итого:          790
Комментарий:    Без лука

Заказ:
Мангал: Шашлык * 2
Пицца: Маргарита * 1
```
````

Without an address (pickup):
````
```
Номер заказа:   15
Имя:            Иван
Телефон:        +79991234567
Доставка:       Нет
Оплата:         онлайн
Итого:          640
Комментарий:    

Заказ:
Мангал: Шашлык * 2
```
````

Rules:
- The text starts with the line ```` ``` ```` and ends with ```` ``` ```` with no newline after the closing fence.
- Every line ends with `\n`. Each item line is `{Category}: {Name} * {Amount}\n`.
- `Итого` with delivery is `Total + Delivery`; for pickup it is `Total`.
- An empty comment leaves `Комментарий:    ` followed by nothing (four spaces after the colon stay).
- To keep the Markdown code block intact, replace every backtick in `Name`, `Address`, `Comment`, item `Category` and item `Name` with `'`.

## Tests to write
- `TestSendMessage`: `httptest.NewServer` checks the path `/bot<token>/sendMessage`, method, headers and the three body fields; answers `{"ok":true}` → nil.
- `TestSendMessageClientError`: 400 with `{"ok":false,"description":"chat not found"}` → error text `telegram: chat not found`.
- `TestSendMessageServerError`: 502 → error containing `status 502`.
- `TestSendMessageNoTokenInError`: point the client at a closed server; the returned error text must not contain the token (wrap or rewrite the `*url.Error` so that the URL is not included).
- `TestFormatOrderDelivery`: the input matching the first example gives exactly that string (write the expected value as a Go string with explicit `\n`).
- `TestFormatOrderPickup`: the second example, exactly.
- `TestFormatOrderFractions`: `Total` 59950 and `Delivery` 15050 print `599.50`, `150.50`, and `Итого:          750`.
- `TestFormatOrderBackticks`: a comment containing a backtick is printed with `'`.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Deciding when to send (task 31). Retrying failed sends.
