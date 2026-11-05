# Citypark API — frontend reference

This file describes the endpoints that are implemented. Every task that adds or changes an endpoint updates it in the same commit (see `docs/tasks/CONVENTIONS.md`, "API documentation").

## General

- Base path: `/api/v1`. Requests and responses are JSON (`Content-Type: application/json`).
- Auth: `Authorization: Bearer <token>`. Staff use JWTs from `POST /auth/sign-in`; customer sessions use `cs_` tokens when `CUSTOMER_AUTH_ENABLED=true`.
- With customer auth enabled, `/user`, `/addresses`, `/orders/my`, and `/orders/{id}` accept customer sessions; legacy customer JWTs return 401. Existing admin JWT access stays available where it was allowed. `POST /orders` stays open to guests, but any supplied invalid credential returns 401 before an order is written.
- Roles: `public` (no token), `user` (any signed-in user, admins included), `admin`.
- Money is in rubles as a JSON number (`599` or `5.99`). Timestamps are RFC 3339.
- Paged lists take `page` (from 1) and `pageSize` (1–500).

## Errors

Every error has the same shape. `errors` is present only for validation failures; its keys are the JSON paths of the invalid fields.

```json
{
  "statusCode": 400,
  "message": "Ошибка валидации, пожалуйста проверьте введенные вами данные",
  "errors": { "phone": "телефон должен быть действительным номером телефона", "items[0].amount": "значение должно быть больше 0" }
}
```

| Status | Meaning | Default message |
|---|---|---|
| 400 | malformed JSON or validation failure | `Некорректный JSON` / `Ошибка валидации, ...` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the role is not allowed | `Недостаточно прав` |
| 404 | not found | `Не найдено` |
| 500 | server error | `Внутренняя ошибка сервера` |

## Endpoints

Sections are grouped by area, in this order: Health, Auth, User, Addresses, Delivery prices, Catalog, Hotel, Rooms, News, Content, Settings, Working hours, Orders, Reports, Integrations, Uploads. Add an area heading (`### Auth`) when its first endpoint is documented. Each endpoint uses this format:

````markdown
#### POST /addresses

Adds a delivery address to the signed-in user.

- **Role:** user

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `city` | string | yes | at most 80 characters |
| `address` | string | yes | at most 1000 characters |

```json
{ "city": "Сочи", "address": "ул. Ленина, 1" }
```

**Response 201**

```json
{ "id": 1, "city": "Сочи", "address": "ул. Ленина, 1", "createdAt": "2024-05-24T12:00:00Z", "updatedAt": "2024-05-24T12:00:00Z" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | a field is missing or too long | validation message, see `errors` |
| 401 | no token | `Требуется авторизация` |

**Notes**

Posting the same address twice returns the same `id`.
````

Leave out "Request body" when there is none, list query and path parameters in a "Parameters" table with the same columns, and leave out "Notes" when there is nothing the frontend needs to know.

<!-- Endpoint sections start below this line. -->

### Health

#### GET /health

Reports that the server is up.

- **Role:** public

**Response 200**

```json
{ "status": "ok" }
```

### Auth

#### POST /auth/sign-up

Registers a new customer account.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `name` | string | yes | at most 80 characters |
| `email` | string | no | a valid email address; stored in lower case |
| `phone` | string | no | a Russian phone number in any common format; stored as `+7XXXXXXXXXX`; absent becomes `""` in profile JSON |
| `password` | string | yes | 8 to 64 characters |

```json
{ "name": "Иван", "email": "ivan@example.com", "phone": "8 (999) 123-45-67", "password": "secret-pass" }
```

**Response 201**

```json
{}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a field is missing or invalid | validation message, see `errors` |

**Notes**

With customer auth disabled, the response is always `201 {}`, also when the email or phone is already registered. Multiple accounts may omit phone. The server does not reveal whether an account exists, so no token is returned: call `POST /auth/sign-in` afterwards. When customer auth is enabled, this legacy customer registration returns 503 `Вход временно недоступен` because verified registration uses `/auth/customer/register`. `is_email_confirmed` is not exposed in JSON. Responses use `Cache-Control: private, no-store`.

#### POST /auth/sign-in

Signs a user in and returns a token.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `email` | string | yes | case-insensitive |
| `password` | string | yes | |

```json
{ "email": "ivan@example.com", "password": "secret-pass" }
```

**Response 200**

```json
{ "id": 1, "name": "Иван", "email": "ivan@example.com", "phone": "+79991234567", "role": "user", "token": "<jwt>" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | `email` or `password` is missing | validation message, see `errors` |
| 401 | unknown email or wrong password | `неверный логин или пароль.` |
| 503 | customer account while customer auth is enabled | `Вход временно недоступен` |

**Notes**

Send the token as `Authorization: Bearer <token>` on later requests. This route keeps issuing staff JWTs. When customer auth is enabled, it stops issuing customer JWTs because verified customers use `/auth/customer/sign-in`. Unknown email and wrong password give the same 401. Responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/google/start

Creates a ten-minute Google authorization attempt. It does not sign in or link an account yet.

- **Role:** public login; recent customer web session for linking

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `intent` | string | yes | `login` or `link` |

```json
{"intent":"login"}
```

**Response 200**

```json
{"authorizationUrl":"https://accounts.google.com/o/oauth2/v2/auth?client_id=client-id&code_challenge=...&code_challenge_method=S256&nonce=...&redirect_uri=https%3A%2F%2Fsite.example%2Faccount%2Fgoogle%2Fcallback&response_type=code&scope=openid+email+profile&state=...","flowToken":"<random>","expiresAt":"2026-10-29T12:10:00Z"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | invalid intent | `Некорректный способ входа` |
| 401 | linking without a valid customer session, or invalid supplied credential | `Требуется авторизация` |
| 403 | linking from a Mini App session | `Откройте сайт для подключения Google` |
| 403 | linking with a web session authenticated more than five minutes ago | `Подтвердите вход повторно` |
| 429 | more than 20 starts per direct IP in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | Google or customer authentication disabled | `Вход через Google недоступен` |

**Notes**

Store `flowToken` in an HttpOnly same-site cookie on the Next server. The backend stores only its hash. Send the browser to `authorizationUrl`; it requests only `openid email profile`, uses state and nonce, and uses PKCE S256. The callback must match `GOOGLE_CALLBACK_URL` exactly. Complete the attempt with the endpoint below. All responses, including errors, use `Cache-Control: private, no-store`. 429 includes `Retry-After` in seconds.

#### POST /auth/customer/google/complete

Consumes a browser-bound Google attempt, exchanges the authorization code on the server, validates the ID token and signs in by Google subject. A link attempt attaches the subject only to the account bound at start.

- **Role:** public for login; the same recent customer web session used at start for linking

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `state` | string | yes | State returned by Google, 32 random bytes in base64url |
| `flowToken` | string | yes | Browser cookie value from start, 32 random bytes in base64url |
| `code` | string | conditional | Authorization code; exactly one of `code` or `error` |
| `error` | string | conditional | `access_denied` for cancellation; exactly one of `code` or `error` |

```json
{"state":"<state>","flowToken":"<flowToken>","code":"<authorization-code>"}
```

Cancellation: `{"state":"<state>","flowToken":"<flowToken>","error":"access_denied"}`. A valid cancellation consumes the attempt.

**Response 200 (login)**

```json
{"user":{"id":1,"name":"Google User","email":"","phone":"","role":"user"},"token":"cs_<random>","expiresAt":"2026-11-29T12:00:00Z"}
```

**Response 200 (link)**

```json
{"linked":true}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | invalid, expired or replayed proof, invalid code or ID token | `Не удалось подтвердить вход через Google` |
| 400 | valid cancellation | `Вход через Google отменён` |
| 401 | linking without the bound valid customer session, or invalid supplied credential | `Требуется авторизация` |
| 403 | linking from a Mini App session | `Откройте сайт для подключения Google` |
| 403 | stale or different web session | `Подтвердите вход повторно` |
| 409 | Google subject owned by another account, or another Google subject already linked | `Способ входа уже используется` |
| 429 | more than 20 completions per direct IP in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | Google or customer authentication disabled, or provider unavailable | `Вход через Google недоступен` |

**Notes**

All responses, including errors, use `Cache-Control: private, no-store`; 429 includes `Retry-After` in seconds. The attempt is consumed before exchange, so failures require a new start. Google email, even when verified, does not become an authentication email or merge key. Keep the login token on the Next server in an HttpOnly same-site cookie; a successful link retains the current session.

#### POST /auth/customer/register

Creates a pending password account and sends a verification link.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `name` | string | yes | at most 80 characters |
| `email` | string | yes | valid email; trimmed and lowercased |
| `phone` | string | no | Russian phone; normalized to `+7XXXXXXXXXX` |
| `password` | string | yes | 8–64 characters |

```json
{"name":"Иван","email":"ivan@example.com","password":"example-password"}
```

**Response 202**

```json
{"message":"Если адрес доступен, письмо с подтверждением отправлено"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or fields | `Некорректный JSON` or validation message with `errors` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

Duplicates, occupied contact email or phone, failed delivery and send limits return the same 202. Existing credentials are not changed. Verification links open `/account/verify?token=...`; opening the link does not consume the proof. All responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/verification/resend

Sends a new proof to a pending email account; invalidates its previous proof.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `email` | string | yes | valid email; trimmed and lowercased |

```json
{"email":"ivan@example.com"}
```

**Response 202**

```json
{"message":"Если адрес доступен, письмо с подтверждением отправлено"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or email | `Некорректный JSON` or validation message with `errors` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

Unknown, verified, limited and delivery failure cases return the same 202. Send limits are 3 per email and 10 per direct IP per hour. All responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/verification/confirm

Consumes a 24-hour verification proof and enables password sign-in. It does not create a session.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `token` | string | yes | proof from verification email |

```json
{"token":"<proof>"}
```

**Response 200**

```json
{"verified":true}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or missing token | `Некорректный JSON` or validation message with `errors` |
| 400 | invalid, expired or used proof | `Ссылка недействительна или устарела` |
| 429 | more than 20 attempts per direct IP in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

429 includes `Retry-After` in seconds. All responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/password/forgot

Requests a password recovery link for an existing verified email/password account.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `email` | string | yes | valid email; trimmed and lowercased |

```json
{"email":"ivan@example.com"}
```

**Response 202**

```json
{"message":"Если аккаунт найден, письмо для восстановления отправлено"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or email | `Некорректный JSON` or validation message with `errors` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

Unknown, admin, unverified, provider-only, send-limited and delivery-failure cases return the same 202. Sends share the 3/email/hour and 10/direct-IP/hour budget with registration and verification. The one-hour link opens `/account/reset?token=...`; GET does not consume the proof. All responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/password/reset

Consumes a recovery proof, replaces the password and revokes every customer session. No session is issued.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `token` | string | yes | recovery proof from email |
| `password` | string | yes | 8–64 characters |

```json
{"token":"<proof>","password":"new-example-password"}
```

**Response 204:** empty body.

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or fields | `Некорректный JSON` or validation message with `errors` |
| 400 | invalid, expired, wrong-purpose or used proof, or changed credential | `Ссылка недействительна или устарела` |
| 429 | more than 20 attempts per direct IP in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

429 includes `Retry-After` in seconds. Reset revokes all existing customer sessions, invalidates outstanding reset proofs and requires a new sign-in. All responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/sign-in

Checks a verified password credential and creates a 30-day web session.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `email` | string | yes | valid email; trimmed and lowercased |
| `password` | string | yes | password |

```json
{"email":"ivan@example.com","password":"example-password"}
```

**Response 200**

```json
{"user":{"id":1,"name":"Иван","email":"ivan@example.com","phone":"","role":"user"},"token":"cs_<random>","expiresAt":"2026-11-28T12:00:00Z"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or fields | `Некорректный JSON` or validation message with `errors` |
| 401 | unknown, wrong, unverified or admin credential | `неверный логин или пароль.` |
| 429 | more than 10 attempts per email or 30 per direct IP in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

429 includes `Retry-After` seconds. Use `Authorization: Bearer cs_<random>` on later requests. All responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/telegram

Validates raw Telegram Mini App `initData` with the dedicated customer bot and issues a 30-day mini session. It never merges accounts based on name, email or phone.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `initData` | string | yes | raw encoded `Telegram.WebApp.initData`, at most 16 KiB; never `initDataUnsafe` |

```json
{"initData":"auth_date=1793361600&query_id=q+%2B+%26%3D+%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82&signature=third-party-signature&user=%7B%22id%22%3A123%2C%22first_name%22%3A%22%D0%A2%D0%B5%D1%81%D1%82%22%7D&hash=b5365ffc37925bd7b51716c98e8167487e8401c2f088ea3c81de1f617faee5b8"}
```

**Response 200**

```json
{"user":{"id":1,"name":"Тест","email":"","phone":"","role":"user"},"token":"cs_<random>","expiresAt":"2026-11-29T12:00:00Z"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON or over 16 KiB | `Некорректный JSON` |
| 401 | invalid, expired or replayed proof | `Не удалось подтвердить вход через Telegram` |
| 429 | more than 30 attempts per direct IP in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth or dedicated bot token disabled | `Вход через Telegram недоступен` |

**Notes**

Send a fresh launch after a lost response; replaying a proof never issues another session. The `signature` field, when present, is included in the bot-token HMAC. 429 includes `Retry-After` seconds. All responses use `Cache-Control: private, no-store`.

#### POST /customer/email/change

Stages a new email for an existing password account and sends a proof. The current email remains usable.

- **Role:** customer web session authenticated within 5 minutes

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `email` | string | yes | valid email; trimmed and lowercased |

```json
{"email":"new@example.com"}
```

**Response 202**

```json
{"message":"Если адрес доступен, письмо с подтверждением отправлено"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or email | `Некорректный JSON` or validation message with `errors` |
| 401 | missing or invalid session | `Требуется авторизация` |
| 403 | stale, non-web or ineligible account | `Подтвердите вход повторно` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

An occupied email, send limit and failed delivery give the same 202. Proof links open `/account/email?token=...` and require a separate POST to confirm. All responses use `Cache-Control: private, no-store`.

#### POST /customer/email/confirm

Consumes a change proof, switches the email identity and revokes all other sessions.

- **Role:** same customer web account authenticated within 5 minutes

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `token` | string | yes | proof from change email |

```json
{"token":"<proof>"}
```

**Response 200**

```json
{"verified":true}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid JSON or missing token | `Некорректный JSON` or validation message with `errors` |
| 400 | invalid, expired, used or another account's proof | `Ссылка недействительна или устарела` |
| 401 | missing or invalid session | `Требуется авторизация` |
| 403 | stale or non-web session | `Подтвердите вход повторно` |
| 409 | email was reserved before confirmation | `Способ входа уже используется` |
| 429 | more than 20 confirmations per direct IP in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

The 409 transaction rolls back proof consumption, so the user may retry. 429 includes `Retry-After` seconds. All responses use `Cache-Control: private, no-store`.

#### POST /auth/customer/logout

Revokes the current customer session.

- **Role:** customer session

**Response 204**

No response body.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | missing, malformed, expired or revoked session | `Требуется авторизация` |
| 503 | customer authentication is disabled | `Вход временно недоступен` |

**Notes**

Send `Authorization: Bearer cs_<token>`. Tokens have a fixed 30-day lifetime. Responses and errors use `Cache-Control: private, no-store`.

### User

#### POST /customer/pairings

Creates a five-minute pairing code from a recently authenticated Telegram Mini App session. Replaces an earlier pending code from this account.

- **Role:** customer mini session with Telegram method, authenticated within five minutes

**Request body:** none.

**Response 201**

```json
{"id":"<22-character-id>","code":"<22-character-code>","expiresAt":"2026-10-30T12:05:00Z"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | absent or invalid session | `Требуется авторизация` |
| 403 | stale authentication | `Подтвердите вход повторно` |
| 403 | wrong surface or method | `Подтвердите связь в нужном интерфейсе` |
| 429 | over five creates per account per hour | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

The code is returned only here. Do not put it in a URL. All pairing responses and errors use `Cache-Control: private, no-store`; 429 includes `Retry-After` seconds.

#### POST /customer/pairings/redeem

Binds an unused code to the website account and the exact web session. Redemption does not confirm either side.

- **Role:** customer web session authenticated within five minutes

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `code` | string | yes | 22-character code from Mini App; send only in POST body |

```json
{"code":"<22-character-code>"}
```

**Response 200**

```json
{"id":"<22-character-id>","status":"pending","sourceConfirmed":false,"destinationConfirmed":false,"otherName":"Mini","expiresAt":"2026-10-30T12:05:00Z"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | invalid, expired, canceled or already redeemed code | `Код недействителен или устарел` |
| 400 | malformed JSON | `Некорректный JSON` |
| 401 | absent or invalid session | `Требуется авторизация` |
| 403 | stale authentication | `Подтвердите вход повторно` |
| 403 | wrong surface | `Подтвердите связь в нужном интерфейсе` |
| 409 | both sessions belong to one account | `Аккаунты уже связаны` |
| 429 | over 10 attempts per direct IP or 5 per account in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth disabled | `Вход временно недоступен` |

#### GET /customer/pairings/{id}

Returns the pending or completed pairing to a bound participant. The ID alone grants no access.

- **Role:** participating recent customer session

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | string | yes | pairing ID from create or redeem |

**Response 200**

```json
{"id":"<22-character-id>","status":"pending","sourceConfirmed":false,"destinationConfirmed":false,"otherName":"Website","expiresAt":"2026-10-30T12:05:00Z"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | expired pairing | `Код недействителен или устарел` |
| 401 | absent or invalid session | `Требуется авторизация` |
| 403 | stale authentication | `Подтвердите вход повторно` |
| 403 | wrong surface | `Подтвердите связь в нужном интерфейсе` |
| 404 | unknown ID or unrelated session | `Не найдено` |
| 503 | customer auth disabled | `Вход временно недоступен` |

#### POST /customer/pairings/{id}/confirm

Confirms the calling side. When both sides confirm, ownership and identities move to the website account in one transaction.

- **Role:** participating recent customer session

**Parameters:** `id`, pairing ID from create or redeem.

**Request body**

```json
{}
```

**Response 200**

```json
{"id":"<22-character-id>","status":"completed","sourceConfirmed":true,"destinationConfirmed":true,"otherName":"Mini","expiresAt":"2026-10-30T12:05:00Z"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | expired, canceled or unredeemed pairing; bound proof expired or revoked | `Код недействителен или устарел` |
| 400 | malformed JSON | `Некорректный JSON` |
| 401 | absent or invalid session | `Требуется авторизация` |
| 403 | stale authentication | `Подтвердите вход повторно` |
| 403 | wrong surface or forbidden merge | `Подтвердите связь в нужном интерфейсе` or `Объединение аккаунтов недоступно` |
| 404 | unknown ID or unrelated session | `Не найдено` |
| 409 | incompatible credentials or methods | `Способ входа уже используется` |
| 429 | over 20 confirmations per account in 15 minutes | `Слишком много попыток. Попробуйте позже` |
| 503 | customer auth disabled | `Вход временно недоступен` |

**Notes**

A retry from the valid website session returns `completed` without merging again. The source session is revoked at completion. The Mini App must relaunch through Telegram to see the merged account.

Saved addresses merge in the same transaction. Destination metadata wins for normalized city/address duplicates (the smallest destination association ID wins when legacy duplicates exist); existing destination duplicates remain visible. Unique source entries retain their association IDs, names, structured fields, review state and creation times. Source duplicates retain the smallest source ID. The destination default wins; otherwise the surviving equivalent of the source default becomes default. If neither book has a default, none is assigned. Shared address rows and historical order delivery snapshots are unchanged. Any failure rolls back the entire merge and pairing consumption.

#### DELETE /customer/pairings/{id}

Cancels a pending pairing. Canceling an already canceled or completed pairing is idempotent.

- **Role:** participating customer session

**Parameters:** `id`, pairing ID from create or redeem.

**Response 204:** empty body.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | absent or invalid session | `Требуется авторизация` |
| 403 | wrong surface | `Подтвердите связь в нужном интерфейсе` |
| 404 | unknown ID or unrelated session | `Не найдено` |
| 503 | customer auth disabled | `Вход временно недоступен` |

#### GET /customer/methods

Lists the current customer's linked login methods in email, Google, Telegram order. Unlinked methods are omitted. This read-only endpoint does not link or unlink accounts. Google linking uses `POST /auth/customer/google/start` with `intent: "link"` and its callback; Telegram linking uses the two-sided `/customer/pairings` flow. Provider email, subject, IDs and tokens are never returned.

- **Role:** customer session
- **Request:** no parameters or body

```http
GET /api/v1/customer/methods
Authorization: Bearer cs_<customer-session>
```

**Response 200**

```json
{"methods":[{"provider":"email","verified":true},{"provider":"telegram","verified":true}]}
```

An account without identities returns `{"methods":[]}`. The email flag reflects email confirmation; linked Google and Telegram proofs are verified when attached.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | missing, invalid, expired or revoked session | `Требуется авторизация` |
| 403 | staff JWT | `Недостаточно прав` |
| 503 | customer authentication is disabled | `Вход временно недоступен` |

Success and errors use `Cache-Control: private, no-store`.

#### GET /customer/me

Returns the customer profile.

- **Role:** customer session

**Response 200**

```json
{"id":1,"name":"Иван","email":"","phone":"+79990000001","role":"user"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | missing, malformed, expired or revoked session | `Требуется авторизация` |
| 503 | customer authentication is disabled | `Вход временно недоступен` |

**Notes**

Email and phone are empty strings when absent. Staff JWTs and legacy customer JWTs are rejected. Responses and errors use `Cache-Control: private, no-store`.

#### PUT /customer/me

Updates the customer name and optional phone without changing authentication credentials.

- **Role:** customer session

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `name` | string | yes | 1–80 characters |
| `phone` | string | no | Russian phone; normalized to `+7XXXXXXXXXX`; omitted preserves it; empty clears it |

```json
{"name":"Новое имя","phone":"8 999 000 00 03"}
```

**Response 200**

```json
{"id":1,"name":"Новое имя","email":"","phone":"+79990000003","role":"user"}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | invalid name or phone | validation message, see `errors` |
| 400 | email, password, role or another unknown field supplied | `Недопустимые поля профиля` |
| 401 | missing or invalid session | `Требуется авторизация` |
| 409 | phone belongs to another user | `Телефон уже используется` |
| 503 | customer authentication is disabled | `Вход временно недоступен` |

**Notes**

Responses and errors use `Cache-Control: private, no-store`.

#### GET /user

Returns the signed-in user's profile.

- **Role:** user

**Response 200**

```json
{ "id": 1, "name": "user", "email": "", "phone": "+79990000001", "role": "user" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token, invalid token, or the user no longer exists | `Требуется авторизация` |

**Notes**

`email` and `phone` are empty strings when absent. Authentication email is separate from editable profile details. Responses use `Cache-Control: private, no-store`.

#### PUT /user

Updates the signed-in user's name and phone. Changing the authentication email requires a future verification flow.

- **Role:** user

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `name` | string | yes | at most 80 characters |
| `email` | string | no | if supplied, must equal the current email after trimming and lowercasing; empty or omitted preserves it |
| `phone` | string | no | a Russian phone number in any common format; stored as `+7XXXXXXXXXX`; empty or omitted keeps the current phone |

```json
{ "name": "Новое имя", "email": "New@Example.com", "phone": "8 999 000 00 02" }
```

**Response 200**

```json
{ "id": 1, "name": "Новое имя", "email": "new@example.com", "phone": "+79990000002", "role": "user" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | `name` is missing or a field is invalid | validation message, see `errors` |
| 401 | no token, invalid token, or the user no longer exists | `Требуется авторизация` |
| 409 | the email or phone belongs to another user | `email или телефон уже используется` |
| 409 | supplied email differs from the current email | `Для изменения email подтвердите новый адрес` |

**Notes**

Name and phone updates preserve the stored email, password hashes and email confirmation flag. Omitted email and phone preserve their current values. Responses use `Cache-Control: private, no-store`.

### Addresses

#### GET /addresses

Returns the signed-in user's delivery addresses, ordered by `id`.

- **Role:** user

**Response 200**

```json
[
  { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1", "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" }
]
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |

**Notes**

The list is `[]` when the user has no addresses.

#### POST /addresses

Adds a delivery address to the signed-in user.

- **Role:** user

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `city` | string | yes | at most 80 characters; surrounding spaces are removed |
| `address` | string | yes | at most 1000 characters; surrounding spaces are removed |

```json
{ "city": "Сочи", "address": "ул. Ленина, 1" }
```

**Response 201**

```json
{ "id": 1, "city": "Сочи", "address": "ул. Ленина, 1", "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a field is missing, blank or too long | validation message, see `errors` |
| 401 | no token or invalid token | `Требуется авторизация` |

**Notes**

Posting the same address twice returns the same `id`. Addresses are shared between users: another user who posts the same city and address gets the same `id`.

The first link in an empty address book becomes the default. Reposting preserves existing structured metadata. New legacy links require review before checkout autofill; response IDs remain shared address IDs.

#### DELETE /addresses/{id}

Removes an address from the signed-in user's list.

- **Role:** user

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer (path) | yes | a positive address id |

**Response 204**

No body.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 404 | `id` is not a positive integer | `Не найдено` |

**Notes**

The answer is 204 also when the address does not exist or is not in the user's list. Only the user's link is removed; other users keep the address. Deleting the default leaves no default; no replacement is selected. Shared addresses and order history are preserved.

### Customer saved addresses

All responses, including errors, send Cache-Control: private, no-store. With CUSTOMER_AUTH_ENABLED=true, active web/password or Google and Mini App/Telegram cs_ sessions work; absent, invalid, revoked or expired credentials and all legacy/staff JWTs return 401. With the flag false, a legacy user JWT works, staff returns 403, invalid/unknown-role JWT returns 401, and any cs_ flow returns 503. Authenticated non-user identities reaching the group return 403. Owner is derived only from authentication. Saved IDs identify user-address associations, unlike legacy /addresses shared IDs. Address storage does not prove delivery eligibility. Unexpected errors never include submitted fields in logs or monitoring.

Entry fields: `id`, `name`, `city`, `address`, `street`, `apartment`, `entrance`, `isDefault`, `needsReview`, `createdAt`, `updatedAt`. All are required in responses; timestamps are RFC 3339. `address` is immutable full text. Internal address/user IDs and credentials are never exposed. Legacy entries keep full text verbatim, with empty `street` and `needsReview=true`; they must be corrected before checkout autofill, even when default. Lists are ordered by ascending saved ID and empty lists contain `[]`.

#### GET /customer/addresses

List saved addresses.

- **Role:** customer user; authentication matrix above applies.

No request body is required.

**Response 200**

```json
{"addresses":[{"id":1,"name":"Дом","city":"Белореченск","address":"ул. Гоголя, 61, кв. 12, подъезд 2, этаж 3","street":"ул. Гоголя, 61","apartment":"12","entrance":"2, этаж 3","isDefault":true,"needsReview":false,"createdAt":"2026-11-05T13:07:41.090029Z","updatedAt":"2026-11-05T13:07:41.090029Z"},{"id":2,"name":"","city":"Город","address":"  старая строка  ","street":"","apartment":"","entrance":"","isDefault":false,"needsReview":true,"createdAt":"2026-11-05T13:07:41.092927Z","updatedAt":"2026-11-05T13:07:41.092927Z"}]}
```

**Errors**

| Status | Message / condition |
|---|---|
| 401 | Требуется авторизация (including an owner that became merged/unavailable). |
| 403 | Недостаточно прав. |
| 503 | Вход временно недоступен. |
| 500 | Внутренняя ошибка сервера. |

#### POST /customer/addresses

Create or reuse a saved address.

- **Role:** customer user; authentication matrix above applies.

**Request body**

Only name, city, street, apartment and entrance are accepted, each as a non-null string. Unknown fields (including userId, id and isDefault), null/scalar/array documents, multiple documents, non-string fields and bodies larger than 1 MiB return 400 Некорректный JSON. Optional name/apartment/entrance default to empty strings. Unicode boundary whitespace is trimmed; limits count code points. City and street are required. Default changes use the separate route.

| Field | Limits | Exact field error |
|---|---|---|
| name | at most 40 | `Не больше 40 символов` |
| city | required, at most 80 | empty: `Выберите населённый пункт`; too long: `Не больше 80 символов` |
| street | 3–120 | short: `Укажите улицу и дом`; long: `Слишком длинный адрес` |
| apartment | at most 20 | `Не больше 20 символов` |
| entrance | at most 20 | `Не больше 20 символов` |

```json
{"name":"Дом","city":"Белореченск","street":"ул. Гоголя, 61","apartment":"12","entrance":"2, этаж 3"}
```

**Response 201**

```json
{"entry":{"id":1,"name":"Дом","city":"Белореченск","address":"ул. Гоголя, 61, кв. 12, подъезд 2, этаж 3","street":"ул. Гоголя, 61","apartment":"12","entrance":"2, этаж 3","isDefault":true,"needsReview":false,"createdAt":"2026-11-05T13:07:41.135643Z","updatedAt":"2026-11-05T13:07:41.135643Z"},"created":true}
```

**Response 200**

```json
{"entry":{"id":1,"name":"Дом","city":"Белореченск","address":"ул. Гоголя, 61, кв. 12, подъезд 2, этаж 3","street":"ул. Гоголя, 61","apartment":"12","entrance":"2, этаж 3","isDefault":true,"needsReview":false,"createdAt":"2026-11-05T13:07:41.135643Z","updatedAt":"2026-11-05T13:07:41.135643Z"},"created":false}
```

**Errors**

| Status | Message / condition |
|---|---|
| 400 | Malformed/type/unknown-field/null/extra-document/body-too-large: Некорректный JSON. Invalid fields: Проверьте адрес, with exact errors per field. |
| 401 | Требуется авторизация (including an owner that became merged/unavailable). |
| 403 | Недостаточно прав. |
| 503 | Вход временно недоступен. |
| 500 | Внутренняя ошибка сервера. |

Duplicates are matched by NFC, collapsed Unicode whitespace and lowercase city/full text; punctuation and apartment/entrance remain significant. Creation returns the lowest matching saved ID with 200 and created=false, preserving name/default/timestamps and legacy review state. No fields are overwritten. A new entry returns 201 and created=true; only the first link in an empty book becomes default.

#### PUT /customer/addresses/{id}

Replace editable saved-address fields.

- **Role:** customer user; authentication matrix above applies.
- **Parameter:** `id`, positive int64 saved-entry ID belonging to the authenticated user.

**Request body**

Only name, city, street, apartment and entrance are accepted, each as a non-null string. Unknown fields (including userId, id and isDefault), null/scalar/array documents, multiple documents, non-string fields and bodies larger than 1 MiB return 400 Некорректный JSON. Optional name/apartment/entrance default to empty strings. Unicode boundary whitespace is trimmed; limits count code points. City and street are required. Default changes use the separate route.

| Field | Limits | Exact field error |
|---|---|---|
| name | at most 40 | `Не больше 40 символов` |
| city | required, at most 80 | empty: `Выберите населённый пункт`; too long: `Не больше 80 символов` |
| street | 3–120 | short: `Укажите улицу и дом`; long: `Слишком длинный адрес` |
| apartment | at most 20 | `Не больше 20 символов` |
| entrance | at most 20 | `Не больше 20 символов` |

```json
{"name":"Дом","city":"Белореченск","street":"ул. Гоголя, 61","apartment":"12","entrance":"2, этаж 3"}
```

**Response 200**

```json
{"entry":{"id":1,"name":"","city":"Город","address":"Дом 3","street":"Дом 3","apartment":"","entrance":"","isDefault":true,"needsReview":false,"createdAt":"2026-11-05T13:07:41.181725Z","updatedAt":"2026-11-05T13:07:41.187934Z"}}
```

**Errors**

| Status | Message / condition |
|---|---|
| 400 | Malformed/type/unknown-field/null/extra-document/body-too-large: Некорректный JSON. Invalid fields: Проверьте адрес, with exact errors per field. |
| 401 | Требуется авторизация (including an owner that became merged/unavailable). |
| 403 | Недостаточно прав. |
| 404 | Не найдено (invalid, nonpositive, overflow, absent or foreign saved-entry ID). |
| 409 | Такой адрес уже сохранён; errors.street has the same text. |
| 503 | Вход временно недоступен. |
| 500 | Внутренняя ошибка сервера. |

Replaces all editable fields; omitted optional fields become empty strings. Preserves saved ID, creation timestamp and default flag. Explicitly editing a legacy entry clears needsReview. Shared address rows and order snapshots are never rewritten. Collision with another entry returns 409 without changes.

#### PUT /customer/addresses/{id}/default

Select the default saved address.

- **Role:** customer user; authentication matrix above applies.
- **Parameter:** `id`, positive int64 saved-entry ID belonging to the authenticated user.

No request body is required.

**Response 200**

```json
{"entry":{"id":3,"name":"","city":"Город","address":"Дом 4","street":"Дом 4","apartment":"","entrance":"","isDefault":true,"needsReview":false,"createdAt":"2026-11-05T13:07:41.190288Z","updatedAt":"2026-11-05T13:07:41.192173Z"}}
```

**Errors**

| Status | Message / condition |
|---|---|
| 401 | Требуется авторизация (including an owner that became merged/unavailable). |
| 403 | Недостаточно прав. |
| 404 | Не найдено (invalid, nonpositive, overflow, absent or foreign saved-entry ID). |
| 503 | Вход временно недоступен. |
| 500 | Внутренняя ошибка сервера. |

Selection is idempotent and returns the fresh entry. Missing/foreign IDs leave the old default unchanged. Legacy entries may be selected but still require review before checkout autofill.

#### DELETE /customer/addresses/{id}

Delete a saved-address association.

- **Role:** customer user; authentication matrix above applies.
- **Parameter:** `id`, positive int64 saved-entry ID belonging to the authenticated user.

No request body is required.

**Response 204**

No body.

**Errors**

| Status | Message / condition |
|---|---|
| 401 | Требуется авторизация (including an owner that became merged/unavailable). |
| 403 | Недостаточно прав. |
| 404 | Не найдено (invalid, nonpositive, overflow, absent or foreign saved-entry ID). |
| 503 | Вход временно недоступен. |
| 500 | Внутренняя ошибка сервера. |

Absent/foreign entries return 404. Deleting a default leaves no default; shared addresses, other owners and orders survive.

### Delivery prices

#### GET /delivery-prices

Returns the delivery price of every town, ordered by `id`.

- **Role:** public

**Response 200**

```json
[
  { "id": 1, "name": "Белореченск", "value": "", "price": 100, "freeFrom": 500, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" },
  { "id": 2, "name": "п. Родники", "value": "", "price": 250, "freeFrom": 0, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" }
]
```

**Notes**

The list is `[]` when there are no delivery prices. An order is matched to a price by comparing the address `city` with `name`. `freeFrom` is the order total (rubles, without delivery) from which delivery to this town is free; `0` means never free. After `make migrate` the list holds the 58 towns of the old site.

#### POST /delivery-prices

Creates one or more delivery prices.

- **Role:** admin

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `deliveryPrices` | array | yes | at least one item |
| `deliveryPrices[].name` | string | yes | at most 255 characters; surrounding spaces are removed; must be unique |
| `deliveryPrices[].value` | string | no | at most 255 characters; surrounding spaces are removed |
| `deliveryPrices[].price` | number | yes | rubles, greater than 0 |
| `deliveryPrices[].freeFrom` | number | no | rubles, 0 or more; order total from which delivery is free; `0` (default) = never free |

```json
{ "deliveryPrices": [{ "name": "Белореченск", "price": 100, "freeFrom": 500 }, { "name": "п. Родники", "price": 250 }] }
```

**Response 201**

```json
[
  { "id": 1, "name": "Белореченск", "value": "", "price": 100, "freeFrom": 500, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" },
  { "id": 2, "name": "п. Родники", "value": "", "price": 250, "freeFrom": 0, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" }
]
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | the array is empty, or a field is missing, blank or out of range | validation message, see `errors` (keys look like `deliveryPrices[0].name`) |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 409 | a name is already taken, or repeated in the request | `name должно быть уникальным` |

**Notes**

All items are created or none: when one item fails, nothing is saved.

#### PATCH /delivery-prices/{id}

Replaces the name, description, price and free-delivery threshold of a delivery price.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer (path) | yes | a positive delivery price id |

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `name` | string | yes | at most 255 characters; surrounding spaces are removed; must be unique |
| `value` | string | no | at most 255 characters; surrounding spaces are removed |
| `price` | number | yes | rubles, greater than 0 |
| `freeFrom` | number | no | rubles, 0 or more; `0` (default) = never free |

```json
{ "name": "Белореченск", "value": "", "price": 120, "freeFrom": 500 }
```

**Response 200**

```json
{ "id": 1, "name": "Белореченск", "value": "", "price": 120, "freeFrom": 500, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:05:10.1182Z" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a field is missing, blank or out of range | validation message, see `errors` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | no delivery price with this `id`, or `id` is not a positive integer | `Не найдено` |
| 409 | the name is used by another delivery price | `name должно быть уникальным` |

**Notes**

This is a full replacement of the three fields: an omitted `value` becomes an empty string.

#### DELETE /delivery-prices/{id}

Deletes a delivery price.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer (path) | yes | a positive delivery price id |

**Response 204**

No body.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | `id` is not a positive integer | `Не найдено` |

**Notes**

The answer is 204 also when the delivery price does not exist.

### Catalog

#### GET /categories

Returns every category that is not deleted and not hidden in iiko, ordered by sort order and then by `id`.

- **Role:** public

**Response 200**

```json
[
  {
    "id": 1,
    "name": "Мангал",
    "description": "",
    "images": [{ "id": 1, "url": "https://example.com/a.jpg", "type": "from_iiko" }],
    "isAvailableForOrder": true,
    "salesDays": "Пн, Вт, Ср, Чт, Пт",
    "startTime": "9:00",
    "endTime": "21:00"
  }
]
```

**Notes**

The list is `[]` when there are no categories. `isAvailableForOrder` is computed on the server for the current moment in the restaurant's time zone: it is `true` when today is one of `salesDays` and the time is inside `startTime`–`endTime`. A category without a sale window is always available, and its `salesDays`, `startTime` and `endTime` are empty strings. A category hidden in iiko is not listed, but `GET /categories/{id}` and its products still work.

#### GET /categories/{id}

Returns one category.

- **Role:** public

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer (path) | yes | a positive category id |

**Response 200**

```json
{
  "id": 1,
  "name": "Мангал",
  "description": "",
  "images": [{ "id": 1, "url": "https://example.com/a.jpg", "type": "from_iiko" }],
  "isAvailableForOrder": true,
  "salesDays": "Пн, Вт, Ср, Чт, Пт",
  "startTime": "9:00",
  "endTime": "21:00"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 404 | `id` is not a positive integer, or the category does not exist or is deleted | `Не найдено` |

#### GET /categories/{id}/products

Returns the products of one category, ordered by sort order and then by `id`. Deleted products are left out.

- **Role:** public

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer (path) | yes | a positive category id |

**Response 200**

```json
[
  {
    "id": 1,
    "categoryId": 1,
    "name": "Шашлык из баранины",
    "description": "",
    "measureUnit": "",
    "weight": 0,
    "proteins": 0,
    "fats": 0,
    "carbohydrates": 0,
    "calories": 0,
    "flags": 0,
    "price": 320,
    "images": [],
    "isAvailableForOrder": true,
    "salesDays": "",
    "startTime": "",
    "endTime": ""
  }
]
```

**Errors**

| Status | When | Message |
|---|---|---|
| 404 | `id` is not a positive integer | `Не найдено` |

**Notes**

An unknown or deleted category gives `200` with `[]`, not `404`. `price` is in rubles. `isAvailableForOrder` is `true` only when `isOrderBlocked` is false and both the product's own sale window and its category's window are open now; `salesDays`, `startTime` and `endTime` describe the product's own window and are empty strings when it has none.

#### PUT /products/{id}/ordering-block

Role: admin (Bearer JWT). Required path `id`: product ID. Required JSON boolean `isOrderBlocked`; true blocks ordering until manually unblocked, false removes only the manual block.

Request example: `{"isOrderBlocked":true}`.

Success: 200, the updated Product object (same fields as GET /products, including images), with `isOrderBlocked: true` and `isAvailableForOrder: false` when blocked.

Errors: 401 without valid authentication, 403 for a non-admin, 404 (`Не найдено`) for invalid, missing or deleted IDs. Invalid JSON/types return the standard 400 decode error. Missing/null flag returns 400 with message `Ошибка валидации, пожалуйста проверьте введенные вами данные` and `errors.isOrderBlocked` = `Укажите, заблокирован ли заказ блюда`.

Product responses include boolean `isOrderBlocked` (default false). Blocked products remain visible in category and product listings. `isAvailableForOrder` requires both an absent manual block and open product/category schedules. iiko sync preserves the manual block; unblocking never overrides schedules.

POST /orders rejects a blocked product with 503 and `Заказ блюда «<name>» временно недоступен.`, including duplicate retries before returning a payment link. It does not silently omit items or cancel existing orders/payments.

#### GET /products

Searches products across all categories. Without `query` it returns every product.

- **Role:** public

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `query` | string (query) | no | words separated by spaces; case does not matter; only the first 20 words are used |
| `page` | integer (query) | no | at least 1, default 1 |
| `pageSize` | integer (query) | no | from 1 to 500; the server has a default |

**Response 200**

```json
[
  {
    "id": 1,
    "categoryId": 1,
    "name": "Шашлык из баранины",
    "description": "",
    "measureUnit": "",
    "weight": 0,
    "proteins": 0,
    "fats": 0,
    "carbohydrates": 0,
    "calories": 0,
    "flags": 0,
    "price": 320,
    "images": [],
    "isAvailableForOrder": true,
    "salesDays": "",
    "startTime": "",
    "endTime": ""
  }
]
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | `page` or `pageSize` is not an integer of at least 1, or `pageSize` is above 500 | `Некорректные параметры пагинации` |

**Notes**

A product matches when any of the words matches its name, its category name or one of its tags. `?query=хачапури+салат` and `?query=хачапури%20салат` both give two words. Results are ordered by sort order and then by `id`. The list is `[]` when nothing matches or the page is past the end.

### Hotel

#### GET /hotel/rooms

Returns how many hotel rooms of each type are free right now.

- **Role:** public

**Response 200**

```json
{
  "rooms": { "vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0 },
  "free": 4,
  "updatedAt": "2026-10-22T12:00:00Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 404 | no data yet: the hotel's sender has not posted the counts since the table was created | `Не найдено` |

**Notes**

404 means no data yet; the site hides the counter. `free` is the sum of the five counts. The keys of `rooms`, the website room ids and the maximum number of free rooms are:

| JSON key | Website room id | Rooms | Max |
|---|---|---|---|
| `vip` | `vip` | № 301 | 1 |
| `singleBalcony` | `single-balcony` | № 201 | 1 |
| `single` | `single` | № 204, 304 | 2 |
| `doubleTwin` | `double-twin` | № 203, 206, 302, 303 | 4 |
| `doubleEuro` | `double-euro` | № 202, 205, 305, 306 | 4 |

#### POST /hotel/rooms

Replaces the free-room counts of all five room types. Meant for the hotel's external sender, not the frontend.

- **Role:** public, protected by a shared secret in the `X-Webhook-Secret` header

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `X-Webhook-Secret` | string (header) | yes | must equal the server's `HOTEL_WEBHOOK_SECRET` |

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `rooms` | object | yes | holds all five counts |
| `rooms.vip` | integer | yes | 0–1 |
| `rooms.singleBalcony` | integer | yes | 0–1 |
| `rooms.single` | integer | yes | 0–2 |
| `rooms.doubleTwin` | integer | yes | 0–4 |
| `rooms.doubleEuro` | integer | yes | 0–4 |

```json
{ "rooms": { "vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0 } }
```

**Response 200**

The same body as `GET /hotel/rooms`.

```json
{
  "rooms": { "vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0 },
  "free": 4,
  "updatedAt": "2026-10-22T12:00:00Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a count is missing or out of range | validation message, see `errors` (for example `rooms.vip`: `значение должно быть не больше 1`, `rooms.doubleEuro`: `поле не должно быть пустым` or `значение должно быть не меньше 0`) |
| 401 | the header is missing or does not match the secret | `Требуется авторизация` |
| 404 | the server has no `HOTEL_WEBHOOK_SECRET` configured, so the endpoint is disabled | `Не найдено` |

**Notes**

Meant for the hotel's external sender, not the frontend; send all five types every time. A missing type is an error, not "unchanged", and `0` is a valid count. The old three-counter body (`double_rooms`, `single_rooms`, `vip_room`) is not accepted.

#### GET /hotel/prices

Returns the hotel fees and the breakfast menu.

- **Role:** public

**Response 200**

Money is in rubles. `breakfast` is in the order the admin saved it and is `[]` when the menu is empty.

```json
{
  "breakfastDelivery": 100,
  "lateCheckoutPerHour": 500,
  "smokingFine": 2000,
  "breakfast": [
    { "name": "Омлет, 130 г", "shortName": "Омлет", "price": 250, "desktopOnly": false },
    { "name": "Сырники", "shortName": "", "price": 300, "desktopOnly": false },
    { "name": "Блины с творогом", "shortName": "", "price": 200, "desktopOnly": true },
    { "name": "Круассан с сёмгой", "shortName": "", "price": 350, "desktopOnly": false },
    { "name": "Каша с фруктами", "shortName": "", "price": 80, "desktopOnly": true }
  ],
  "updatedAt": "2026-10-22T21:30:03.595242Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 404 | the prices row was deleted from the database (does not happen normally: the migration creates it) | `Не найдено` |

**Notes**

These numbers fill the placeholders `{{breakfastDelivery}}`, `{{lateCheckoutPerHour}}` and `{{smokingFine}}` of the content blocks. `desktopOnly` items are hidden on the mobile version of the site. `shortName` is `""` when the item has no short form.

#### PUT /hotel/prices

Replaces the three fees and the whole breakfast menu.

- **Role:** admin (`Authorization: Bearer <token>`)

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `breakfastDelivery` | number | yes | rubles, 0–100000, at most 2 decimals; `0` is valid |
| `lateCheckoutPerHour` | number | yes | rubles, 0–100000, at most 2 decimals |
| `smokingFine` | number | yes | rubles, 0–100000, at most 2 decimals |
| `breakfast` | array | no | 0–30 items; a missing field means an empty menu |
| `breakfast[].name` | string | yes | at most 80 characters, surrounding spaces are removed |
| `breakfast[].shortName` | string | no | at most 40 characters, surrounding spaces are removed |
| `breakfast[].price` | number | yes | rubles, 0–100000, at most 2 decimals |
| `breakfast[].desktopOnly` | boolean | no | default `false` |

`updatedAt` is not accepted.

```json
{
  "breakfastDelivery": 150,
  "lateCheckoutPerHour": 600.5,
  "smokingFine": 3000,
  "breakfast": [
    { "name": "Яичница", "shortName": "Яйца", "price": 199.99, "desktopOnly": true },
    { "name": "Омлет", "price": 0 }
  ]
}
```

**Response 200**

The stored state, the same as `GET /hotel/prices`.

```json
{
  "breakfastDelivery": 150,
  "lateCheckoutPerHour": 600.5,
  "smokingFine": 3000,
  "breakfast": [
    { "name": "Яичница", "shortName": "Яйца", "price": 199.99, "desktopOnly": true },
    { "name": "Омлет", "shortName": "", "price": 0, "desktopOnly": false }
  ],
  "updatedAt": "2026-10-22T21:30:03.644761Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a field is missing, blank, too long or out of range | validation message, see `errors` (keys look like `smokingFine`, `breakfast[0].name`, `breakfast[1].price`, `breakfast`; for example `поле не должно быть пустым`, `значение должно быть не меньше 0`, `значение должно быть не больше 80`, `значение должно быть не больше 30` for more than 30 items) |
| 400 | a price has more than 2 decimals | `errors` key of the field: `не более 2 знаков после запятой` |
| 401 | no token or a bad token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |

**Notes**

This is a full replacement in one transaction: send the whole breakfast list every time. The items are stored in the order sent, so the order of the array is the order on the site. On any error nothing changes. These numbers fill the placeholders `{{breakfastDelivery}}`, `{{lateCheckoutPerHour}}` and `{{smokingFine}}` of the content blocks.

### Rooms

#### GET /rooms

Returns the five hotel room types with their texts, price per night and photo, ordered as on the site (`vip`, `single-balcony`, `single`, `double-twin`, `double-euro`).

- **Role:** public

**Response 200**

```json
[
  {
    "id": "vip",
    "kind": "vip",
    "balcony": true,
    "name": "VIP-номер",
    "hotelName": "VIP, 2 комнаты",
    "numbers": "№ 301",
    "area": "45 + 7 м²",
    "areaDetailed": "45 м² + балкон 7 м²",
    "capacity": "до 4 гостей",
    "summary": "Гостиная и спальня, расширенная комплектация, балкон над кронами парка.",
    "description": "<p>Гостиная и спальня, расширенная комплектация, балкон над кронами парка.</p>",
    "pricePerNight": 8000,
    "photo": { "url": "/images/room-vip.jpg", "alt": "VIP-номер", "position": "" },
    "gallery": [
      { "url": "/images/room-vip-living.jpg", "alt": "VIP-номер: гостиная" },
      { "url": "/images/room-vip-bathroom.jpg", "alt": "VIP-номер: санузел" },
      { "url": "/images/room-vip-balcony.jpg", "alt": "VIP-номер: балкон" }
    ],
    "hotelNote": "№ 301 · до 4 гостей",
    "updatedAt": "2026-10-22T21:15:35.527937Z"
  },
  {
    "id": "single",
    "kind": "single",
    "balcony": false,
    "name": "Одноместный",
    "hotelName": "",
    "numbers": "№ 204, 304",
    "area": "19 м²",
    "areaDetailed": "",
    "capacity": "",
    "summary": "",
    "description": "",
    "pricePerNight": 4000,
    "photo": { "url": "/images/room-single.jpg", "alt": "Одноместный номер", "position": "right center" },
    "gallery": [],
    "hotelNote": "№ 204, 304 · двуспальная",
    "updatedAt": "2026-10-22T21:15:35.527937Z"
  }
]
```

The example shows two of the five items.

**Notes**

The five room ids are fixed (the same as in `GET /hotel/rooms`): there is no way to add or remove a room type. `pricePerNight` is rubles for one night, the same all year. `summary` — одна-две фразы для карточки на сайте, простой текст, до 200 символов; `description` — полный текст (HTML). Optional texts that were never filled in are empty strings, not `null`. `description` is HTML with `p`, `br`, `strong`, `em`, `a`, `ul`, `ol`, `li`; anything else is removed on save. Plain texts use normal spaces; the site adds non-breaking spaces itself. `gallery` — дополнительные фото для страницы номера, в заданном порядке; `photo` остаётся главным фото карточки. `gallery` is always an array (`[]` when empty, never `null`); `gallery[].url` follows the same rules as `photo.url` (an uploaded image comes back as a full URL). `photo.url` is either a site path (`/images/...`) or an uploaded image: an uploaded image comes back as a full URL (the server's public base URL plus `/uploads/...`).

#### PUT /rooms/{id}

Replaces all the editable fields of one room type. The room price is edited only here.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | string (path) | yes | one of `vip`, `single-balcony`, `single`, `double-twin`, `double-euro` |

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `name` | string | yes | at most 100 characters |
| `hotelName` | string | no | at most 100 characters |
| `numbers` | string | yes | at most 100 characters |
| `area` | string | yes | at most 100 characters |
| `areaDetailed` | string | no | at most 100 characters |
| `capacity` | string | no | at most 100 characters |
| `summary` | string | no | plain text, at most 200 characters, stored as sent; a missing field is stored as `""` |
| `description` | string | no | HTML, cleaned on save; at most 1000 characters of visible text (tags do not count) |
| `pricePerNight` | number | yes | rubles, greater than 0, at most 1000000, at most 2 decimals |
| `photo` | object | yes | |
| `photo.url` | string | yes | a site path `/images/...`, an uploaded image (`/uploads/...` or the full URL that `POST /uploads` returned) |
| `photo.alt` | string | no | at most 200 characters |
| `photo.position` | string | no | `""` or `«X% Y%»` with X and Y from 0 to 100, for example `50% 30%` |
| `gallery` | array | no | at most 12 items; `url` as in `photo.url`; `alt` at most 200 characters; a missing field or `null` is stored as `[]` |
| `gallery[].url` | string | yes | a site path `/images/...`, an uploaded image (`/uploads/...` or the full URL that `POST /uploads` returned) |
| `gallery[].alt` | string | no | at most 200 characters |
| `hotelNote` | string | no | at most 100 characters |

The gallery replaces the old one as a whole: send every photo you want to keep, in the order to show. Uploaded files that are no longer used by `photo` or `gallery` are deleted from the server. Surrounding spaces of the plain-text fields are removed before the checks. `id`, `kind`, `balcony` and `updatedAt` are not accepted: they cannot be changed.

```json
{
  "name": "Новое имя",
  "hotelName": "Для отеля",
  "numbers": "№ 1, 2",
  "area": "30 м²",
  "areaDetailed": "30 м² + балкон",
  "capacity": "до 3 гостей",
  "summary": "Кратко о номере",
  "description": "<p>Описание</p>",
  "pricePerNight": 5200,
  "photo": { "url": "/images/room-single.jpg", "alt": "Номер", "position": "50% 30%" },
  "gallery": [
    { "url": "/images/room-vip-living.jpg", "alt": "Гостиная" },
    { "url": "/images/room-vip-balcony.jpg", "alt": "Балкон" }
  ],
  "hotelNote": "№ 1, 2 · кровать"
}
```

**Response 200**

The room as in `GET /rooms`, with the stored values.

```json
{
  "id": "vip",
  "kind": "vip",
  "balcony": true,
  "name": "Новое имя",
  "hotelName": "Для отеля",
  "numbers": "№ 1, 2",
  "area": "30 м²",
  "areaDetailed": "30 м² + балкон",
  "capacity": "до 3 гостей",
  "summary": "Кратко о номере",
  "description": "<p>Описание</p>",
  "pricePerNight": 5200,
  "photo": { "url": "/images/room-single.jpg", "alt": "Номер", "position": "50% 30%" },
  "gallery": [
    { "url": "/images/room-vip-living.jpg", "alt": "Гостиная" },
    { "url": "/images/room-vip-balcony.jpg", "alt": "Балкон" }
  ],
  "hotelNote": "№ 1, 2 · кровать",
  "updatedAt": "2026-10-22T21:15:51.390535Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a field is missing, blank, too long or out of range | validation message, see `errors` (keys look like `name`, `pricePerNight`, `photo.url`) |
| 400 | `pricePerNight` has more than 2 decimals | `errors.pricePerNight`: `не более 2 знаков после запятой` |
| 400 | `photo.url` is not a site path or an uploaded image | `errors.photo.url`: `недопустимый адрес фото` |
| 400 | more than 12 `gallery` items | `errors.gallery`: `значение должно быть не больше 12` |
| 400 | a `gallery` item has no `url`, or its `alt` is too long | `errors["gallery[0].url"]`: `поле не должно быть пустым`; `errors["gallery[0].alt"]`: `значение должно быть не больше 200` |
| 400 | `gallery[].url` is not a site path or an uploaded image | `errors["gallery[0].url"]`: `недопустимый адрес фото` (the number is the 0-based index of the item) |
| 400 | `photo.position` is not `""` or `«X% Y%»` | `errors.photo.position`: `формат «50% 30%»` |
| 400 | `description` has more than 1000 characters of visible text | `errors.description`: `не более 1000 символов` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | no room with this `id` | `Не найдено` |

Example of a 400 body:

```json
{
  "statusCode": 400,
  "message": "Ошибка валидации, пожалуйста проверьте введенные вами данные",
  "errors": { "name": "поле не должно быть пустым", "photo.position": "формат «50% 30%»" }
}
```

**Notes**

This is a full replacement: an omitted optional field becomes an empty string. All invalid fields are reported together in `errors`. `description` is sanitized: tags other than `p`, `br`, `strong`, `em`, `a`, `ul`, `ol`, `li` are removed (their text stays), so the response shows what was really stored. When `photo.url` changes, the previous uploaded file is deleted from the server; sending the same photo again keeps it. The seeded `photo.position` values `right center` and `center 70%` do not match the `«X% Y%»` format, so an admin form must not send them back unchanged: send `""` or a value like `50% 30%`.

### News

#### GET /news

Returns a page of news items, newest first (`date` descending, then `id` descending). The list has no `body`: open an item with `GET /news/{slug}` to get it.

- **Role:** public

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `page` | integer (query) | no | from 1, the default is 1 |
| `pageSize` | integer (query) | no | from 1 to 500, the default is the server's page size |

**Response 200**

```json
{
  "news": [
    {
      "id": 1,
      "slug": "special-offer-september-2026",
      "date": "2026-09-07",
      "title": "Спецпредложение сентября",
      "shortTitle": "",
      "excerpt": "Что входит, цена и срок действия.",
      "photo": { "url": "", "alt": "Спецпредложение сентября", "position": "" },
      "updatedAt": "2026-10-22T21:20:07.42121Z"
    },
    {
      "id": 2,
      "slug": "special-offer-june-2026",
      "date": "2026-06-24",
      "title": "Спецпредложение июня",
      "shortTitle": "",
      "excerpt": "",
      "photo": { "url": "", "alt": "", "position": "" },
      "updatedAt": "2026-10-22T21:20:07.42121Z"
    }
  ],
  "count": 7
}
```

The example is `?pageSize=2`. `count` is the number of all items, not of this page.

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | `page` or `pageSize` is not a number, is below 1, or `pageSize` is above 500 | `Некорректные параметры пагинации` |

**Notes**

The list has no `body`. A page past the end gives `"news": []` and the real `count`. Photo is optional: `photo.url` `""` means no photo. An uploaded photo comes back as a full URL (the server's public base URL plus `/uploads/...`); a photo shipped with the site comes back as `/images/...`.

#### GET /news/{slug}

Returns one news item with its `body`.

- **Role:** public

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `slug` | string (path) | yes | the `slug` of an item |

**Response 200**

```json
{
  "id": 4,
  "slug": "new-menu-2026",
  "date": "2026-03-16",
  "title": "Новое меню в Сити Парке 2026",
  "shortTitle": "Новое меню 2026",
  "excerpt": "",
  "body": "",
  "photo": { "url": "/images/news-menu.jpg", "alt": "", "position": "" },
  "updatedAt": "2026-10-22T21:20:07.42121Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 404 | no item with this `slug` | `Не найдено` |

**Notes**

`body` is HTML (see the allowlist in `POST /news`). It is `""` for an item without a text, for example the seeded ones.

#### POST /news

Creates a news item. It is live at once: there are no drafts.

- **Role:** admin

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `title` | string | yes | at most 200 characters |
| `slug` | string | no | `""` makes it from the title; otherwise at most 100 characters, only `a-z`, `0-9` and single `-` between them (`только латиница, цифры и дефисы`) |
| `date` | string | no | `""` is today; otherwise `ГГГГ-ММ-ДД`, a real date (`дата в формате ГГГГ-ММ-ДД`) |
| `shortTitle` | string | no | at most 100 characters |
| `excerpt` | string | no | at most 300 characters |
| `body` | string | no | HTML, cleaned on save; at most 50 000 characters of visible text (tags do not count) |
| `photo` | object | no | |
| `photo.url` | string | no | `""` (no photo), a site path `/images/...`, or an uploaded image (`/uploads/...` or the full URL that `POST /uploads` returned) |
| `photo.alt` | string | no | at most 200 characters |
| `photo.position` | string | no | `""` or `«X% Y%»` with X and Y from 0 to 100, for example `50% 30%` |

Surrounding spaces of the plain-text fields are removed before the checks.

```json
{
  "slug": "",
  "date": "2026-10-20",
  "title": "Новые стейки",
  "shortTitle": "Стейки",
  "excerpt": "Три новых стейка.",
  "body": "<p>Текст <strong>новости</strong></p><script>x</script>",
  "photo": { "url": "/images/news-steaks.jpg", "alt": "Стейк", "position": "50% 30%" }
}
```

**Response 201**

The item as in `GET /news/{slug}`, with the stored values.

```json
{
  "id": 8,
  "slug": "novye-steyki",
  "date": "2026-10-20",
  "title": "Новые стейки",
  "shortTitle": "Стейки",
  "excerpt": "Три новых стейка.",
  "body": "<p>Текст <strong>новости</strong></p>",
  "photo": { "url": "/images/news-steaks.jpg", "alt": "Стейк", "position": "50% 30%" },
  "updatedAt": "2026-10-22T21:20:16.206067Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a field is missing, blank, too long or out of range | validation message, see `errors` (keys look like `title`, `excerpt`, `photo.alt`) |
| 400 | `slug` has other characters | `errors.slug`: `только латиница, цифры и дефисы` |
| 400 | `date` is not `ГГГГ-ММ-ДД` or is not a real date | `errors.date`: `дата в формате ГГГГ-ММ-ДД` |
| 400 | `photo.url` is not a site path or an uploaded image | `errors.photo.url`: `недопустимый адрес фото` |
| 400 | `photo.position` is not `""` or `«X% Y%»` | `errors.photo.position`: `формат «50% 30%»` |
| 400 | `body` has more than 50 000 characters of visible text | `errors.body`: `не более 50000 символов` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 409 | the `slug` sent by the admin is already used | `Новость с таким адресом уже есть` |

Example of a 400 body:

```json
{
  "statusCode": 400,
  "message": "Ошибка валидации, пожалуйста проверьте введенные вами данные",
  "errors": {
    "date": "дата в формате ГГГГ-ММ-ДД",
    "photo.url": "недопустимый адрес фото",
    "slug": "только латиница, цифры и дефисы",
    "title": "поле не должно быть пустым"
  }
}
```

**Notes**

Slug: an empty `slug` is made from the title (Russian letters are transliterated, `Новые стейки` → `novye-steyki`; at most 80 characters; `news` when nothing is left). If a generated slug is taken, `-2`, `-3`, … is added and the first free one is used, so a generated slug never gives 409. A slug typed by the admin that is already used gives 409. The slug is the page address (`/news/{slug}`), so changing it breaks old links. Body: `body` is sanitized, only `p`, `h2`, `h3`, `strong`, `em`, `a`, `ul`, `ol`, `li`, `blockquote`, `br` stay (other tags are removed, the text of `script`, `style` and `iframe` too), so the response shows what was really stored. Photo is optional: `url` `""` means no photo. All invalid fields are reported together in `errors`.

#### PUT /news/{id}

Replaces all the fields of one news item.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer (path) | yes | the `id` of an item |

**Request body**

The same fields and rules as `POST /news`.

```json
{
  "slug": "novye-steyki",
  "date": "2026-10-20",
  "title": "Новые стейки",
  "shortTitle": "Стейки",
  "excerpt": "Три новых стейка.",
  "body": "<p>Текст <strong>новости</strong></p>",
  "photo": { "url": "/images/news-steaks.jpg", "alt": "Стейк", "position": "50% 30%" }
}
```

**Response 200**

The item as in `GET /news/{slug}`, with the stored values.

```json
{
  "id": 8,
  "slug": "novye-steyki",
  "date": "2026-10-20",
  "title": "Новые стейки",
  "shortTitle": "Стейки",
  "excerpt": "Три новых стейка.",
  "body": "<p>Текст <strong>новости</strong></p>",
  "photo": { "url": "/images/news-steaks.jpg", "alt": "Стейк", "position": "50% 30%" },
  "updatedAt": "2026-10-22T21:20:20.100312Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON or an invalid field | the same as `POST /news` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | no item with this `id` | `Не найдено` |
| 409 | the `slug` belongs to another item | `Новость с таким адресом уже есть` |

**Notes**

This is a full replacement: an omitted optional field becomes an empty string (and an omitted `date` becomes today). Send the item's current `slug` to keep it: its own slug is not a conflict. An empty `slug` is made from the title again, which can change the page address. When `photo.url` changes or is cleared, the previous uploaded file is deleted from the server; sending the same photo again keeps it.

#### DELETE /news/{id}

Deletes a news item and its uploaded photo file (a photo shipped with the site, `/images/...`, is not touched).

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer (path) | yes | the `id` of an item |

**Response 204**

No body.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | no item with this `id` (also when it was already deleted) | `Не найдено` |

### Content

Editable page copy is stored as **content blocks**: rows addressed by a block key such as `home.hero.title`. The site reads the blocks of a page; an admin changes the value of a block. Keys are created by the backend; there is no endpoint to add or delete one.

| `type` | `value` in responses | What an admin sends | Rules |
|---|---|---|---|
| `text` | string | `{"value": "строка"}` | trimmed; no line breaks; at most `maxLength` characters (300 unless the block says otherwise) |
| `lines` | array of strings | `{"value": ["a", "b"]}` | each line trimmed, empty ones dropped; 1 to 50 lines; each at most 200 characters |
| `richtext` | string (HTML) | `{"value": "<p>…</p>"}` | cleaned with the *short* profile (`p`, `br`, `strong`, `em`, `a`, `ul`, `ol`, `li`); visible text at most `maxLength` characters (2000 unless the block says otherwise) |
| `image` | `null` (use `image`) | `{"image": {"url", "alt", "position"}}` | `url` required (a site path `/images/...` or an uploaded image from `POST /uploads`); `alt` at most 200 characters; `position` empty or `X% Y%` with X and Y from 0 to 100 |

**Price placeholders.** Never type a price into a text. Write `{{name}}` (spaces inside the braces are allowed) and the site puts the formatted price there. Allowed names (any other name is rejected):

| Placeholder | The site puts there |
|---|---|
| `{{minRoomPrice}}` | the lowest `pricePerNight` of the rooms |
| `{{lateCheckoutPerHour}}` | the late check-out fee per hour (hotel prices, task 52) |
| `{{smokingFine}}` | the smoking fine (hotel prices, task 52) |
| `{{breakfastDelivery}}` | the breakfast delivery fee (hotel prices, task 52) |

Placeholders are stored exactly as typed. They are allowed in `text`, `lines` and `richtext`.

#### GET /content/blocks

Returns the content blocks, ordered by page, then by the block's position, then by key.

- **Role:** public

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `page` | string (query) | no | only the blocks of this page (`home`, `hotel`); all blocks when omitted; an unknown page gives `[]` |

**Response 200**

A JSON array (not wrapped in an object). `value` has the type from the table above; `image` is `null` except for `image` blocks, where `value` is `null`. `maxLength` is the effective limit: the default when the block has none; `0` for `lines` and `image`. `image.url` is a full URL for uploaded images. `section` is the Russian heading the admin groups blocks under.

```json
[
  {
    "key": "hotel.amenities",
    "page": "hotel",
    "section": "Номера",
    "label": "Удобства",
    "type": "lines",
    "maxLength": 0,
    "value": ["Сплит-система", "Smart TV"],
    "image": null,
    "updatedAt": "2026-10-01T10:00:00Z"
  },
  {
    "key": "hotel.intro",
    "page": "hotel",
    "section": "Номера",
    "label": "Вступление",
    "type": "richtext",
    "maxLength": 2000,
    "value": "<p>Привет</p>",
    "image": null,
    "updatedAt": "2026-10-01T10:00:00Z"
  }
]
```

An image block:

```json
{
  "key": "home.photo",
  "page": "home",
  "section": "Первый экран",
  "label": "Фото",
  "type": "image",
  "maxLength": 0,
  "value": null,
  "image": { "url": "http://localhost:3000/uploads/2026/10/fca6521dda98349157f986db0e9751fa.jpg", "alt": "Здание отеля", "position": "50% 30%" },
  "updatedAt": "2026-10-22T21:24:15.872738Z"
}
```

**Notes**

The response is not paged. Replace the placeholders in `text`, `lines` and `richtext` values before showing them.

#### PUT /content/blocks/{key}

Sets the value of an existing block.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `key` | string (path) | yes | the `key` of a block |
| `value` | string or array of strings | for `text`, `lines`, `richtext` | as in the type table; its JSON type must match the block's type |
| `image` | object | for `image` | as in the type table; `value` must be absent or `null` |

**Request body**

```json
{ "value": ["Сплит-система", "Smart TV"] }
```

```json
{ "image": { "url": "/uploads/2026/10/fca6521dda98349157f986db0e9751fa.jpg", "alt": "Здание отеля", "position": "50% 30%" } }
```

**Response 200**

The block as in `GET /content/blocks`, with the stored (cleaned) value.

```json
{
  "key": "hotel.amenities",
  "page": "hotel",
  "section": "Номера",
  "label": "Удобства",
  "type": "lines",
  "maxLength": 0,
  "value": ["Сплит-система", "Smart TV"],
  "image": null,
  "updatedAt": "2026-10-22T21:24:15.87172Z"
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | any invalid value | `Ошибка валидации, пожалуйста проверьте введенные вами данные`, with `errors` (see below) |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | no block with this key | `Не найдено` |

Field errors (`errors`):

| Field | Message |
|---|---|
| `value` | `неверный тип значения` (wrong JSON type for the block, a missing value, or `value` sent to an `image` block) |
| `value` | `не более N символов` (`text` and `richtext`; N is the block's `maxLength`) |
| `value` | `не должно содержать переносов строк` (`text`) |
| `value` | `от 1 до 50 строк` (`lines`) |
| `value` | `каждая строка не более 200 символов` (`lines`) |
| `value` | `строки не должны содержать переносов` (`lines`: one array item contains a line break) |
| `value` | `неизвестная подстановка {{name}}` (a placeholder that is not in the list above) |
| `image` | `неверный тип значения` (an `image` block without the `image` object) |
| `image.url` | `поле не должно быть пустым`; `недопустимый адрес фото` (not `/images/...` or an uploaded image of this server) |
| `image.alt` | `не более 200 символов` |
| `image.position` | `формат «50% 30%»` |

**Notes**

The value is replaced as a whole and nothing is saved when any field is invalid. Send the cleaned text back from the response into the editor. When an `image` block gets a different photo, the previously uploaded file is deleted from the server; sending the same photo again keeps it.

### Settings

The site's contact details are one object. Navigation, legal links, the site name, the chef's name and the copyright year stay in the site's code.

#### GET /settings

Returns the contact details and global promo-code switch.

- **Role:** public

**Response 200**

```json
{
  "promoCodesEnabled": false,
  "phones": {
    "restaurant": { "label": "Ресторан и доставка", "shortLabel": "Ресторан", "display": "8 918 311-97-91", "number": "+79183119791", "altDisplay": "8 (86155) 3-30-03", "altNumber": "+78615533003" },
    "hotel": { "label": "Отель", "shortLabel": "Отель", "display": "8 918 311-97-10", "number": "+79183119710", "altDisplay": "8 (86155) 3-30-02", "altNumber": "+78615533002" },
    "free": { "label": "Бесплатно по России", "shortLabel": "По России", "display": "8 800 100-24-41", "number": "88001002441", "altDisplay": "", "altNumber": "" },
    "landline": { "label": "Городской", "shortLabel": "Городской", "display": "(86155) 2-25-38", "number": "+78615522538", "altDisplay": "", "altNumber": "" }
  },
  "address": { "postcode": "352630", "city": "Белореченск", "street": "ул. Гоголя, 61" },
  "email": "belcitypark@mail.ru",
  "company": { "name": "ООО «Олимп»", "inn": "2309117400", "ogrn": "1092309002378" },
  "social": [
    { "name": "ВКонтакте", "label": "VK", "url": "https://vk.com/cityparkvip", "display": "vk.com/cityparkvip" },
    { "name": "Одноклассники", "label": "OK", "url": "https://ok.ru/cityparkvip", "display": "ok.ru/cityparkvip" }
  ],
  "updatedAt": "2026-10-22T21:26:51.912273Z"
}
```

| Field | Notes |
|---|---|
| `phones.restaurant`, `.hotel`, `.free`, `.landline` | `label` and `shortLabel` are titles; `display` is the text to show; `number` is for the `tel:` link |
| `altDisplay`, `altNumber` | a second (landline) number; both are `""` when there is none |
| `social[].url` | always starts with `https://` |
| `promoCodesEnabled` | Global switch for new promo quotes and orders; defaults to false for existing settings. |

**Errors**

| Status | When | Message |
|---|---|---|
| 404 | the settings row is missing (it is created by a migration, so this should not happen) | `Не найдено` |

**Notes**

The site builds `tel:` links from `number`. The full address line is built by the site from `address`.

#### PUT /settings

Replaces the whole settings object.

- **Role:** admin

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `promoCodesEnabled` | boolean | no | Omission defaults to false. |
| `phones.<name>.label` | string | yes | at most 60 characters (`<name>` is `restaurant`, `hotel`, `free` or `landline`) |
| `phones.<name>.shortLabel` | string | yes | at most 30 characters |
| `phones.<name>.display` | string | yes | at most 40 characters |
| `phones.<name>.number` | string | yes | at most 20 characters; `+` and 10 to 15 digits |
| `phones.<name>.altDisplay` | string | no | at most 40 characters; set together with `altNumber` |
| `phones.<name>.altNumber` | string | no | at most 20 characters; `+` and 10 to 15 digits; set together with `altDisplay` |
| `address.postcode` | string | yes | exactly 6 digits |
| `address.city` | string | yes | at most 60 characters |
| `address.street` | string | yes | at most 100 characters |
| `email` | string | yes | an e-mail address, at most 100 characters |
| `company.name` | string | yes | at most 100 characters |
| `company.inn` | string | yes | 10 to 12 digits |
| `company.ogrn` | string | yes | 13 to 15 digits |
| `social` | array | yes | at most 10 items; `null` is stored as `[]` |
| `social[].name` | string | yes | at most 40 characters |
| `social[].label` | string | yes | at most 10 characters |
| `social[].url` | string | yes | at most 200 characters; starts with `https://` |
| `social[].display` | string | yes | at most 60 characters |

Every string is trimmed before it is checked. An `updatedAt` in the body is ignored.

```json
{
  "promoCodesEnabled": false,
  "phones": {
    "restaurant": { "label": "Ресторан и доставка", "shortLabel": "Ресторан", "display": "8 918 311-97-91", "number": "+79183119791", "altDisplay": "8 (86155) 3-30-03", "altNumber": "+78615533003" },
    "hotel": { "label": "Отель", "shortLabel": "Отель", "display": "8 918 311-97-10", "number": "+79183119710", "altDisplay": "8 (86155) 3-30-02", "altNumber": "+78615533002" },
    "free": { "label": "Бесплатно по России", "shortLabel": "По России", "display": "8 800 100-24-41", "number": "88001002441", "altDisplay": "", "altNumber": "" },
    "landline": { "label": "Городской", "shortLabel": "Городской", "display": "(86155) 2-25-38", "number": "+78615522538", "altDisplay": "", "altNumber": "" }
  },
  "address": { "postcode": "352630", "city": "Белореченск", "street": "ул. Гоголя, 61" },
  "email": "belcitypark@mail.ru",
  "company": { "name": "ООО «Олимп»", "inn": "2309117400", "ogrn": "1092309002378" },
  "social": [
    { "name": "ВКонтакте", "label": "VK", "url": "https://vk.com/cityparkvip", "display": "vk.com/cityparkvip" },
    { "name": "Одноклассники", "label": "OK", "url": "https://ok.ru/cityparkvip", "display": "ok.ru/cityparkvip" }
  ]
}
```

**Response 200**

The stored object as in `GET /settings`, with the new `updatedAt`.

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | any invalid field | `Ошибка валидации, пожалуйста проверьте введенные вами данные`, with `errors` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | the settings row is missing | `Не найдено` |

Field errors (`errors`, besides the general ones in "Errors"):

| Field | Message |
|---|---|
| `phones.<name>.number`, `phones.<name>.altNumber` | `номер: только цифры и «+»` |
| `phones.<name>.altDisplay`, `phones.<name>.altNumber` | `заполните оба поля` (only one of the two is filled; the error is on the empty one) |
| `social[i].url` | `ссылка должна начинаться с https://` |

Example of a 400 response:

```json
{
  "statusCode": 400,
  "message": "Ошибка валидации, пожалуйста проверьте введенные вами данные",
  "errors": {
    "email": "email должен быть электронной почтой",
    "social[0].url": "ссылка должна начинаться с https://"
  }
}
```

**Notes**

PUT replaces everything; send the object you got from GET with your changes. Nothing is saved when any field is invalid.

### Working hours

The hours when orders are accepted are one record. The admin changes them; the site shows them and `POST /orders` checks them.

#### GET /working-hours

Returns the working hours.

- **Role:** public

**Response 200**

```json
{ "start": "09:00", "end": "22:00", "updatedAt": "2026-10-28T17:31:43.999866Z" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 404 | the row is missing (it is created by a migration, so this should not happen) | `Не найдено` |

**Notes**

Times are `HH:MM` with a leading zero; show them with `9:00`-style formatting (no leading zero). Orders are accepted from `start` until `end` (not at `end`). A «Ко времени» `deliveryTime` may equal `end`.

#### PUT /working-hours

Replaces both times.

- **Role:** admin

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `start` | string | yes | `HH:MM`, 24 hours, with a leading zero (`09:00`, not `9:00`); spaces around it are trimmed |
| `end` | string | yes | same format; must be later than `start` (no overnight hours) |

```json
{ "start": "10:00", "end": "23:00" }
```

An `updatedAt` in the body is ignored.

**Response 200**

```json
{ "start": "10:00", "end": "23:00", "updatedAt": "2026-10-28T17:32:08.701197Z" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | malformed JSON | `Некорректный JSON` |
| 400 | a time is empty or not `HH:MM` | `Ошибка валидации, пожалуйста проверьте введенные вами данные`, with `errors.start` or `errors.end` = `Укажите время в формате ЧЧ:ММ` (an empty value gives `поле не должно быть пустым`) |
| 400 | `end` is not later than `start` | `Ошибка валидации, пожалуйста проверьте введенные вами данные`, with `errors.end` = `Время окончания должно быть позже начала` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | the row is missing | `Не найдено` |

Example of a 400 response:

```json
{
  "statusCode": 400,
  "message": "Ошибка валидации, пожалуйста проверьте введенные вами данные",
  "errors": { "end": "Время окончания должно быть позже начала" }
}
```

**Notes**

The new hours apply to the next `POST /orders` at once. Nothing is saved when a field is invalid.

### Orders

#### POST /orders

Places an order. A checkout contact is found by phone number or created; existing profiles and saved addresses are never changed. Orders are accepted only during the restaurant's working hours.

- **Role:** public, with optional Bearer authentication. An invalid, expired, empty or repeated Authorization header returns 401; it never becomes a guest checkout. A deleted user or a token whose role no longer matches also returns 401.

Promo orders require a confirmed quote total. A mismatch returns 409 before any customer/address/order writes. Fetch a fresh quote, show the changed total and require a new submission; do not redirect automatically.

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `promoCode` | string | no | Trimmed, ASCII uppercase `[A-Z0-9_-]{3,32}`; empty means no code |
| `expectedPayableTotal` | number | required with promoCode | Positive rubles; must equal current discounted food + delivery, or the saved amount on duplicate retries |
| `phone` | string | yes | a valid Russian phone number; it is stored as `+7XXXXXXXXXX` |
| `name` | string | yes | at most 80 characters |
| `comment` | string | no | at most 8192 characters |
| `deliveryTime` | string | no | `HH:MM` (24 hours, for example `18:30`); empty or left out means «как можно скорее». The date is always today. The time must be within the working hours from `GET /working-hours`, both ends included |
| `address` | object | no | delivery address; leave it out for pickup |
| `address.city` | string | yes, when `address` is set | at most 80 characters; must be the name of a delivery price |
| `address.address` | string | yes, when `address` is set | at most 1000 characters |
| `items` | array | yes | at least one item |
| `items[].productId` | integer | yes | greater than 0 |
| `items[].amount` | integer | yes | greater than 0 |

```json
{
  "phone": "+79991234567",
  "name": "Иван",
  "comment": "Без лука",
  "deliveryTime": "18:30",
  "address": { "city": "Белореченск", "address": "ул. Ленина, 1" },
  "items": [{ "productId": 1, "amount": 2 }]
}
```

**Response 201**

```json
{ "orderId": 15, "redirectUrl": "https://assistant.demo.moneta.ru/?MNT_AMOUNT=790.00&MNT_CURRENCY_CODE=RUB&MNT_FAIL_URL=http%3A%2F%2Flocalhost%3A3001%2Fcheckout%3Fpayment%3Dfailed&MNT_ID=100&MNT_SIGNATURE=95f0d770cc1e67ea9e8a3a70eb14f6be&MNT_SUCCESS_URL=http%3A%2F%2Flocalhost%3A3001%2Fcheckout%2Fdone&MNT_TEST_MODE=0&MNT_TRANSACTION_ID=3f2b8c1e-9d4a-4e7b-a1c6-5d0e8f7a2b94" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | invalid optional authentication, deleted account or changed role | `Требуется авторизация` |
| 400 | the body is not valid JSON | `Некорректный JSON` |
| 400 | a field is missing or invalid; `errors` lists the fields | `Ошибка валидации, пожалуйста проверьте введенные вами данные` |
| 400 | `deliveryTime` is not `HH:MM`; `errors.deliveryTime` has the text | `Укажите время в формате ЧЧ:ММ` |
| 400 | `deliveryTime` is outside the working hours; `errors.deliveryTime` has the text (built from the current hours) | `Мы работаем с 9:00 до 22:00` |
| 400 | the phone number cannot be normalized | `Неверный номер телефона.` |
| 400 | `address.city` is not a known delivery city | `Указано не существующее имя населенного пункта.` |
| 400 | a product does not exist or is deleted | `Продукт не найден.` |
| 400 | Missing/nonpositive expectedPayableTotal on promo order; errors.expectedPayableTotal | `Подтвердите итоговую сумму заказа.` |
| 400 | Global promo switch disabled for a new order; errors.promoCode | `Промокоды временно отключены.` |
| 400 | Invalid promo code format; errors.promoCode | `Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.` |
| 400 | Unknown/disabled/future/expired code on a fresh order; errors.promoCode | `Промокод недействителен.` |
| 400 | No eligible positive spend; errors.promoCode | `В корзине нет блюд для этого промокода.` |
| 400 | Eligible spend below minimum; errors.promoCode | `Недостаточная сумма подходящих блюд для промокода.` |
| 400 | Invalid or overflowing merged quantity | `Некорректное количество.` |
| 409 | Current or saved payable amount differs from confirmation | `Сумма заказа изменилась. Проверьте итог и повторите оформление.` |
| 500 | Database error or monetary overflow | `Внутренняя ошибка сервера` |
| 503 | New promo order with missing RMS/terminal/payment configuration | `Заказы с промокодом временно недоступны.` |
| 503 | the current time is outside the working hours | `Извините, но мы не можем принять ваш заказ.` followed by a newline and `Заказы принимаются с 9:00 до 22:00` (the hours from `GET /working-hours`) |
| 503 | the product's or its category's weekday limit excludes today | `Возможность заказа "<name>" доступна только в следующие дни недели: Пн, Вт, Ср, Чт, Пт` |
| 503 | the product's or its category's time limit excludes now | `Время заказа "<name>" ограничено с 9:00 до 21:00.` |

**Notes**

Orders are paid only online. The frontend sends the customer to `redirectUrl`, the Moneta payment page; after paying, Moneta sends the customer to `MONETA_SUCCESS_URL` (or `MONETA_FAIL_URL`) with `MNT_TRANSACTION_ID` appended. That value is now the order's random payment UUID (lowercase, with dashes), not the order number: `orderId` in the response stays the numeric id, and the frontend must not read the order number from `MNT_TRANSACTION_ID`. Deploy note: payment links created before the deploy fail at the callback, because old numeric transaction ids are rejected. Repeating an identical order (same checkout name, phone, address, comment, `deliveryTime` and items) while the previous one is still new/pending returns the same `orderId` only within the same authenticated account or the same guest checkout scope. Phone alone never finds an earlier order. `paymentType` is not a field any more: responses do not have it, and a request that still sends it is accepted (the field is ignored). The message of the working-hours error contains a real newline (`\n` in JSON); show it as two lines. Identical orders for different times are separate orders. The same product listed twice is merged into one item. `deliveryPrice` is the town's `price`, or 0 when `total` is at least the town's `freeFrom` (when `freeFrom` is above 0); pickup orders have 0.

Authentication with a customer Bearer token assigns the order to that account regardless of the checkout phone. Admin checkout remains unowned. Guests never acquire ownership, even when their phone matches an account. Contact names/phones and delivery text are saved as snapshots for order responses, notifications and iiko; later profile/address changes do not rewrite them.

Guest checkout uses the server-issued `citypark_guest_checkout` cookie. A first request (or an invalid/expired cookie) issues `Set-Cookie: citypark_guest_checkout=<signed-proof>; Path=/api/v1/orders; Max-Age=86400; HttpOnly; Secure; SameSite=Lax`. Forward that cookie on retries to reuse an unpaid order; without it each request starts a new scope. The cookie expires absolutely after 24 hours, grants no access to order history/details, and is never returned in JSON. Use HTTPS. A server-side frontend must forward the backend cookie explicitly; this task does not add frontend cookie handling. Guest identifiers are stored only as hashes.

All order HTTP responses, including errors, send `Cache-Control: private, no-store`.

Promo code is part of duplicate matching, alongside all existing scope/contact/address/item criteria. Same-code pending retries retain original prices, campaign snapshots and delivery even after campaign edits, disabling or expiry. Different codes and no code are separate orders. Retries check product blocks before returning a payment URL; supplied expectedPayableTotal is compared with the saved amount. New orders use current catalog data and a campaign snapshot locked against admin replacement. No-code callers may omit expectedPayableTotal; if supplied it must match.

All order reads, lists and live events expose subtotal (original food), discount (food discount), promoCode (normalized code or null), total (subtotal minus discount) and per-item discount. Original offeredPrice and quantity are preserved. Historical/no-code orders have zero discount and null promoCode. Promo eligibility is an internal saved boolean, including eligible lines with zero allocation; it is never reconstructed from current campaign targets.

Payment URLs, signed Check responses and Pay amount validation use saved discounted food total plus saved delivery. Duplicate callbacks remain idempotent. Delivery is never discounted; free-delivery thresholds use food after discount.

New promo orders require configured RMS integration. Missing configuration returns 503 before writes/payment links; no-code orders and existing pending same-code retries remain usable with saved pricing.

#### POST /orders/{id}/payment-retry

Obtain a payment link for an existing owned order. **Role:** authenticated customer (website or Mini App session). Guest checkout cookies and administrator credentials cannot use this endpoint.

**Parameters:** `id` is a positive integer path parameter. No request body; no client-supplied amount.

**Request example:** `POST /api/v1/orders/1/payment-retry` with `Authorization: Bearer <customer-session>`.

**Response 200** (captured from `TestPaymentRetryAppWiring`, test merchant configuration):

```json
{"orderId":1,"redirectUrl":"https://demo.moneta.ru/assistant.htm?MNT_AMOUNT=790.00&MNT_CURRENCY_CODE=RUB&MNT_FAIL_URL=http%3A%2F%2Flocalhost%3A3001%2Fcheckout%3Fpayment%3Dfailed&MNT_ID=100&MNT_SIGNATURE=607080d90b8fb471a340eeb0b8e5144a&MNT_SUCCESS_URL=http%3A%2F%2Flocalhost%3A3001%2Fcheckout%2Fdone&MNT_TEST_MODE=0&MNT_TRANSACTION_ID=e8a5f66a-aaba-4938-9e27-cc824aab6024"}
```

**Errors** (standard `{ "statusCode": number, "message": string }` envelope):

| Status | When | Message |
|---|---|---|
| 401 | Missing, invalid or expired credential | `Требуется авторизация` |
| 403 | Administrator or another non-customer role | `Недостаточно прав` |
| 404 | Invalid ID, missing order, unowned guest order or another account's order | `Не найдено` |
| 409 | Status is not new/pending, or payment ID already exists | `Этот заказ уже оплачен или недоступен для оплаты.` |
| 409 | Creation date is not today | `Срок оплаты заказа истёк. Оформите новый заказ.` |
| 503 | Outside restaurant opening hours | `Оплата доступна только в часы работы ресторана.` |
| 500 | Database/hours failure, missing hours configuration or empty generated payment URL | `Внутренняя ошибка сервера` |

**Notes:** Ownership is checked before eligibility. Only `new`/`pending` orders without a payment ID, created today in `APP_TIMEZONE` (default `Europe/Moscow`), are eligible during current opening hours (start inclusive, end exclusive). Future creation dates are also rejected. Uses the original order ID and saved `total + deliveryPrice`, including saved discounts. No repricing, order/status updates, promo consumption or staff notifications occur. Repeated calls sign the same transaction. A link is not a reservation: subsequent Moneta CHECK can reject it after closing or midnight. Payment confirmations for money already collected retain their existing validation and idempotence, even when delayed. All responses, including errors, send `Cache-Control: private, no-store`.

#### GET /orders/my

Lists orders owned by the signed-in account, newest first. Ownership uses `account_id`, independently of the checkout contact or phone. Guest orders are excluded, even when the phone matches. There is no guest-order claiming.

- **Role:** user

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `page` | integer | no | from 1, default 1 |
| `pageSize` | integer | no | 1–500, default is the server's page size |

**Response 200**

```json
{
  "orders": [
    {
      "id": 2,
      "createdAt": "2024-05-24T12:00:00Z",
      "updatedAt": "2024-05-24T12:00:00Z",
      "status": "new",
      "comment": "",
      "deliveryTime": "",
      "total": 599,
      "deliveryPrice": 150,
      "customer": {
        "id": 2,
        "name": "Иван",
        "phone": "+79991234567"
      },
      "address": {
        "id": 1,
        "city": "Сочи",
        "address": "ул. Ленина, 1"
      },
      "items": [
        {
          "productId": 1,
          "name": "Шашлык",
          "categoryName": "Мангал",
          "amount": 1,
          "offeredPrice": 599,
          "discount": 0
        }
      ],
      "subtotal": 599,
      "discount": 0,
      "promoCode": null,
      "paymentRetry": { "allowed": true, "reason": "" }
    }
  ],
  "count": 12
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | `page` or `pageSize` is not an integer of at least 1, or `pageSize` is above 500 | `Некорректные параметры пагинации` |
| 401 | no token or invalid token | `Требуется авторизация` |

**Notes**

Customer responses include `paymentRetry: { "allowed": boolean, "reason": "" | "not_payable" | "expired" | "closed" }`, computed from fresh backend state using the retry policy above. Reasons are evaluated in that order (empty reason means allowed); no payment ID or ownership fields are exposed. An hours lookup needed for eligibility is performed at most once per request. A lookup failure returns 500 `Внутренняя ошибка сервера` rather than guessed availability. Admin detail responses retain their existing order shape without `paymentRetry`. The examples with allowed=true assume a same-day request during opening hours.

`count` is the total number of the user's orders, not the size of the page. `orders` is `[]` when there are none. `address` is `null` for a pickup order. `subtotal` is original food and `total` is `subtotal - discount`, without delivery; add `deliveryPrice` for the amount to pay.

#### GET /orders/{id}

Returns one order.

- **Role:** user

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer | yes | path parameter, the order id |

**Response 200**

```json
{
  "id": 2,
  "createdAt": "2024-05-24T12:00:00Z",
  "updatedAt": "2024-05-24T12:00:00Z",
  "status": "new",
  "comment": "",
  "deliveryTime": "",
  "total": 599,
  "deliveryPrice": 150,
  "customer": {
    "id": 2,
    "name": "Иван",
    "phone": "+79991234567"
  },
  "address": {
    "id": 1,
    "city": "Сочи",
    "address": "ул. Ленина, 1"
  },
  "items": [
    {
      "productId": 1,
      "name": "Шашлык",
      "categoryName": "Мангал",
      "amount": 1,
      "offeredPrice": 599,
      "discount": 0
    }
  ],
  "subtotal": 599,
  "discount": 0,
  "promoCode": null,
  "paymentRetry": { "allowed": true, "reason": "" }
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 404 | the order does not exist, is not a valid id, or belongs to another user | `Не найдено` |

**Notes**

Customer responses include `paymentRetry: { "allowed": boolean, "reason": "" | "not_payable" | "expired" | "closed" }`, computed from fresh backend state using the retry policy above. Reasons are evaluated in that order (empty reason means allowed); no payment ID or ownership fields are exposed. An hours lookup needed for eligibility is performed at most once per request. A lookup failure returns 500 `Внутренняя ошибка сервера` rather than guessed availability. Admin detail responses retain their existing order shape without `paymentRetry`. The examples with allowed=true assume a same-day request during opening hours.

An admin can read any order. A customer can read only an order whose `account_id` matches their authenticated account. Orders owned by another account and unowned guest orders return the same 404 as a missing order, including when their checkout phone matches the customer.

`customer.name`, `customer.phone`, `address.city` and `address.address` are checkout snapshots. `customer.id` is the legacy contact ID, not proof of ownership. Pickup has `address: null`; if the source address was deleted, its snapshot survives with `address.id: 0`. Legacy rows without snapshots fall back to their existing contact/address references. Owner IDs and guest scopes are not exposed. Responses send `Cache-Control: private, no-store`.

**Promo snapshot response example** (captured from TestPromoOrderSnapshot; persisted order fields only, before adding customer `paymentRetry` metadata):

```json
{
  "subtotal": 699,
  "discount": 69.9,
  "promoCode": "SAVE10",
  "id": 1,
  "createdAt": "2026-10-31T12:29:09.237788Z",
  "updatedAt": "2026-10-31T12:29:09.245628Z",
  "status": "pending",
  "comment": "без лука",
  "deliveryTime": "",
  "total": 629.1,
  "deliveryPrice": 150,
  "customer": {
    "id": 1,
    "name": "Пётр",
    "phone": "+79991112233"
  },
  "address": {
    "id": 1,
    "city": "Сочи",
    "address": "Ленина, 1"
  },
  "items": [
    {
      "discount": 59.9,
      "productId": 1,
      "name": "Шашлык",
      "categoryName": "Мангал",
      "amount": 1,
      "offeredPrice": 599
    },
    {
      "discount": 10,
      "productId": 2,
      "name": "Лаваш",
      "categoryName": "Мангал",
      "amount": 1,
      "offeredPrice": 100
    }
  ]
}
```

#### GET /orders/live

Streams every order event over a WebSocket as it happens. The admin's live orders page is built on it.

- **Role:** admin

**Connecting**

A browser cannot set `Authorization` on a WebSocket, so the token travels in the subprotocol header:

```js
const ws = new WebSocket('ws://host/api/v1/orders/live', ['bearer', token])
```

This sends `Sec-WebSocket-Protocol: bearer, <jwt>`. The server answers with the subprotocol `bearer`. The token never goes into the URL.

**Errors** (before the upgrade, as a normal HTTP response)

| Status | When | Message |
|---|---|---|
| 401 | no subprotocol, not exactly `bearer` and a token, or an invalid token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |

**Messages**

Every message is a text frame with one JSON object. `order` has the same shape as the admin response of `GET /orders/{id}`, without customer `paymentRetry` metadata.

| `type` | When |
|---|---|
| `hello` | right after the connection is open; has no `order` |
| `order.pending` | an order was created and waits for payment |
| `order.paid` | an order was paid |

`hello`:

```json
{"type":"hello"}
```

`order.pending`:

```json
{
  "type": "order.pending",
  "order": {
    "id": 2,
    "createdAt": "2024-05-24T12:00:00Z",
    "updatedAt": "2024-05-24T12:00:00Z",
    "status": "pending",
    "comment": "",
    "deliveryTime": "",
    "total": 599,
    "deliveryPrice": 150,
    "customer": {
      "id": 2,
      "name": "Иван",
      "phone": "+79991234567"
    },
    "address": {
      "id": 1,
      "city": "Сочи",
      "address": "ул. Ленина, 1"
    },
    "items": [
      {
        "productId": 1,
        "name": "Шашлык",
        "categoryName": "Мангал",
        "amount": 1,
        "offeredPrice": 599,
        "discount": 0
      }
    ],
    "subtotal": 599,
    "discount": 0,
    "promoCode": null
  }
}
```

`order.paid`:

```json
{
  "type": "order.paid",
  "order": {
    "id": 2,
    "createdAt": "2024-05-24T12:00:00Z",
    "updatedAt": "2024-05-24T12:00:00Z",
    "status": "paid",
    "comment": "",
    "deliveryTime": "",
    "total": 599,
    "deliveryPrice": 150,
    "customer": {
      "id": 2,
      "name": "Иван",
      "phone": "+79991234567"
    },
    "address": {
      "id": 1,
      "city": "Сочи",
      "address": "ул. Ленина, 1"
    },
    "items": [
      {
        "productId": 1,
        "name": "Шашлык",
        "categoryName": "Мангал",
        "amount": 1,
        "offeredPrice": 599,
        "discount": 0
      }
    ],
    "subtotal": 599,
    "discount": 0,
    "promoCode": null
  }
}
```

**Notes**

- The client sends nothing. The server pings every 30 seconds.
- Close code `1013` (try again later) means reconnect: the server is restarting, or the client read too slowly and was dropped.
- Events are not stored and not replayed. After every `hello` (the first connect and each reconnect) the client reloads the list with `GET /orders`.
- A browser cannot read the status of a failed handshake. After a failed connect, make a normal authorised request (for example `GET /orders`) to find out whether the token expired.
- Proxies must pass the `Upgrade` and `Connection` headers and keep idle connections for more than 30 seconds.
- `GET /orders/live` is matched before `GET /orders/{id}`, so `/orders/7` is still an order.

#### GET /orders

Lists orders, newest first. The filters are optional and combine with AND.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `page` | integer | no | from 1, default 1 |
| `pageSize` | integer | no | 1–500, default is the server's page size |
| `q` | string | no | at most 100 characters after trimming. Digits only: the order number or any part of the phone. Otherwise: the customer's name or any part of the phone |
| `from` | string | no | `YYYY-MM-DD`, the first day, inclusive |
| `to` | string | no | `YYYY-MM-DD`, the last day, inclusive, not before `from` |
| `status` | string | no | `new`, `pending`, `paid`, `canceled` or `confirmed` |

Example: `GET /orders?q=%D0%B8%D0%B2%D0%B0%D0%BD&from=2026-10-21&to=2026-10-21&status=paid`.

**Response 200**

```json
{
  "orders": [
    {
      "id": 2,
      "createdAt": "2024-05-24T12:00:00Z",
      "updatedAt": "2024-05-24T12:00:00Z",
      "status": "new",
      "comment": "",
      "deliveryTime": "",
      "total": 599,
      "deliveryPrice": 150,
      "customer": {
        "id": 2,
        "name": "Иван",
        "phone": "+79991234567"
      },
      "address": {
        "id": 1,
        "city": "Сочи",
        "address": "ул. Ленина, 1"
      },
      "items": [
        {
          "productId": 1,
          "name": "Шашлык",
          "categoryName": "Мангал",
          "amount": 1,
          "offeredPrice": 599,
          "discount": 0
        }
      ],
      "subtotal": 599,
      "discount": 0,
      "promoCode": null
    }
  ],
  "count": 12
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | `page` or `pageSize` is not an integer of at least 1, or `pageSize` is above 500 | `Некорректные параметры пагинации` |
| 400 | a search parameter is invalid (see below) | `Некорректные параметры поиска` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |

The `errors` object of a search 400 lists every invalid field:

| Field | Message |
|---|---|
| `q` | `не более 100 символов` |
| `from`, `to` | `дата в формате ГГГГ-ММ-ДД` |
| `to` | `не раньше даты from` |
| `status` | `недопустимое значение` |

```json
{
  "statusCode": 400,
  "message": "Некорректные параметры поиска",
  "errors": {
    "from": "дата в формате ГГГГ-ММ-ДД",
    "status": "недопустимое значение"
  }
}
```

**Notes**

`count` counts the matching orders, not the size of the page. `customer` shows who placed each order.

`q` matches the order number, any part of the customer's phone and the customer's name (case-insensitive). `%` and `_` in `q` are matched literally.

`from` and `to` are local dates (`APP_TIMEZONE`), both inclusive. `from` or `to` alone is allowed. An empty parameter (`?status=`) is the same as a missing one.

Search by name/phone uses checkout snapshots, so editing the contact profile does not change historical matches. Response fields retain their existing shape. Responses send `Cache-Control: private, no-store`.

#### POST /orders/{id}/items

Adds a product to an order, or changes the amount of a product that is already in it.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer | yes | path parameter, the order id |

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `productId` | integer | yes | greater than 0 |
| `amount` | integer | yes | greater than 0 |

```json
{ "productId": 2, "amount": 2 }
```

**Response 200**

The updated order, in the same format as the admin response of `GET /orders/{id}` (without `paymentRetry`).

```json
{
  "id": 2,
  "createdAt": "2024-05-24T12:00:00Z",
  "updatedAt": "2024-05-24T12:00:00Z",
  "status": "new",
  "comment": "",
  "deliveryTime": "",
  "total": 799,
  "deliveryPrice": 150,
  "customer": {
    "id": 2,
    "name": "Иван",
    "phone": "+79991234567"
  },
  "address": {
    "id": 1,
    "city": "Сочи",
    "address": "ул. Ленина, 1"
  },
  "items": [
    {
      "productId": 1,
      "name": "Шашлык",
      "categoryName": "Мангал",
      "amount": 1,
      "offeredPrice": 599,
      "discount": 0
    },
    {
      "productId": 2,
      "name": "Лаваш",
      "categoryName": "Мангал",
      "amount": 2,
      "offeredPrice": 100,
      "discount": 0
    }
  ],
  "subtotal": 799,
  "discount": 0,
  "promoCode": null
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | a field is missing or not greater than 0 | validation message, see `errors` |
| 400 | the product does not exist or was deleted | `Продукт не найден.` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |
| 404 | the order does not exist or `id` is not a valid id | `Не найдено` |

**Notes**

A new item is stored with the product's current price. When the product is already in the order, only `amount` is replaced and the price the item was ordered at (`offeredPrice`) stays as it was. Sale windows are not checked.

Promo orders reject composition changes with HTTP 409 and `Нельзя изменять состав заказа с промокодом.`; no item or monetary snapshot is changed. This also applies to a promo whose allocated discount is zero.

#### DELETE /orders/{id}/items/{productId}

Removes one product from an order.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer | yes | path parameter, the order id |
| `productId` | integer | yes | path parameter, the product id |

**Response 204**

No body.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |
| 404 | `id` or `productId` is not a valid id | `Не найдено` |

**Notes**

The answer is 204 also when the order or the item does not exist, so the request can be repeated safely. The order stays in place even when its last item is removed.

Promo orders reject composition changes with HTTP 409 and `Нельзя изменять состав заказа с промокодом.`; no item or monetary snapshot is changed. This also applies to a promo whose allocated discount is zero.

#### DELETE /orders/{id}

Deletes an order together with its items.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer | yes | path parameter, the order id |

**Response 204**

No body.

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |
| 404 | `id` is not a valid id | `Не найдено` |

**Notes**

The answer is 204 also when the order does not exist, so the request can be repeated safely. The customer and the address are not deleted.

### Reports

The data comes from 1C (the hotel database). The server holds the 1C credentials; the admin panel never calls 1C. 502 means 1C did not answer within 30 s or returned an error. Dates are local (`APP_TIMEZONE`), both inclusive.

#### GET /reports/hotel/sales

Hotel revenue by organization and room for a period.

- **Role:** admin

**Query parameters**

| Parameter | Type | Required | Rules |
|---|---|---|---|
| `from` | date | yes | `YYYY-MM-DD` |
| `to` | date | yes | `YYYY-MM-DD`, not before `from`; the period is at most 366 days |

```
GET /api/v1/reports/hotel/sales?from=2026-10-01&to=2026-10-21
```

**Response 200**

```json
{
  "rows": [
    { "organization": "ИП Иванов", "room": "Стандарт", "count": 2, "sum": 4000 },
    { "organization": "ООО «Олимп»", "room": "VIP-номер", "count": 3, "sum": 12000.5 },
    { "organization": "ООО «Олимп»", "room": "Люкс", "count": 1, "sum": 9000 }
  ],
  "total": { "count": 6, "sum": 25000.5 }
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | `from` or `to` is missing or not `YYYY-MM-DD` | `Некорректные параметры отчёта`, `errors.from` / `errors.to`: `дата в формате ГГГГ-ММ-ДД` |
| 400 | `to` is before `from` | `Некорректные параметры отчёта`, `errors.to`: `не раньше даты from` |
| 400 | the period is longer than 366 days | `Некорректные параметры отчёта`, `errors.to`: `период не больше 366 дней` |
| 401 | no token | `Требуется авторизация` |
| 403 | the user is not an admin | `Недостаточно прав` |
| 404 | 1C is not configured on the server (`ONEC_HOTEL_URL` is empty) | `Не найдено` |
| 502 | 1C did not answer or returned an error | `Сервис отчётов недоступен` |

**Notes**

Rows are sorted by `organization`, then `room`. `count` is a number and may be fractional; `sum` and `total.sum` are rubles. With no data `rows` is `[]` and the total is `0`.

#### GET /reports/hotel/daily

Hotel revenue per day, for a chart.

- **Role:** admin

**Query parameters**

Same as `GET /reports/hotel/sales`.

```
GET /api/v1/reports/hotel/daily?from=2026-10-20&to=2026-10-22
```

**Response 200**

```json
{
  "days": [
    { "date": "2026-10-20", "sum": 8000 },
    { "date": "2026-10-21", "sum": 0 },
    { "date": "2026-10-22", "sum": 100 }
  ],
  "total": 8100
}
```

**Errors**

Same as `GET /reports/hotel/sales`. A date from 1C that cannot be read also gives 502 `Сервис отчётов недоступен`.

**Notes**

`days` has one entry for every calendar day from `from` to `to`, in order, with `0` for days without revenue. `total` is the sum of `days`.

### Integrations

#### POST /iiko/update-products

Downloads the external menu from iiko and applies it: categories and products. Meant to be called by a scheduler or an admin tool, not by the site frontend.

- **Role:** public, protected by a shared secret in the `X-Update-Token` header

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `X-Update-Token` | string (header) | yes | must equal the server's `IIKO_UPDATE_TOKEN` |

**Response 200**

```json
{ "totalCategories": 2, "totalProducts": 27 }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | the header is missing or does not match the token | `Требуется авторизация` |
| 404 | the server has no `IIKO_UPDATE_TOKEN` configured, so the endpoint is disabled | `Не найдено` |
| 409 | another menu sync is running (also one started by the timer or by `POST /iiko/sync`) | `Обновление меню уже выполняется` |
| 500 | iiko could not be reached or the update failed; nothing is changed | `Внутренняя ошибка сервера` |

**Notes**

The update runs in one transaction and can take a few seconds. The menu is the external menu `IIKO_EXTERNAL_MENU_ID`. Every menu group becomes a category, even a group with nothing for sale. Items without a price are skipped. `totalCategories` and `totalProducts` count the categories and products that are in the menu after the update. Items that disappeared from iiko are hidden, not removed. The `isHidden` flag of a menu group is stored; hidden groups are synced with their products but are not listed by `GET /categories`.

Products take their `description` from the iiko menu item (trimmed; an empty string when iiko has none, and an empty iiko description clears the old one on the next update). The update saves each product's iiko button image as a product image of type `from_iiko` (the first size with an image wins; an image url used twice keeps the last product), and deletes `from_iiko` images that are no longer in the menu. Categories get no images from iiko, so their `from_iiko` images are all deleted. The update does not take tags, nutrition values or sale windows from iiko: nutrition values and product sale windows are reset and all tags are removed. Category descriptions are copied from iiko with surrounding whitespace trimmed; an empty description clears the old text. Category ordering times and weekdays are managed by admins and survive sync. Images added by hand stay.

#### POST /iiko/sync

Starts the same menu sync as `POST /iiko/update-products`, for the admin panel's «Обновить меню из iiko» button.

- **Role:** admin

**Parameters**

None. The request has no body.

**Request**

```
POST /api/v1/iiko/sync
Authorization: Bearer <token>
```

**Response 200**

```json
{ "totalCategories": 1, "totalProducts": 1 }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | the token is missing or invalid | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |
| 409 | another menu sync is running, also one started by the timer or by `POST /iiko/update-products` | `Обновление меню уже выполняется` |
| 500 | iiko could not be reached or the sync failed; nothing is changed | `Внутренняя ошибка сервера` |

**Notes**

Takes a few seconds, so show a progress state on the button. Only one sync runs at a time: a second request fails at once with 409 instead of waiting, so show the message and let the user try again later. What the sync writes is described under `POST /iiko/update-products`.

#### POST /webhooks/moneta

Payment notification ("Pay URL") that Moneta calls after an online payment. It is not called by the site frontend; it is documented so that the shop settings in Moneta point at the right address.

- **Role:** public, trusted only when `MNT_SIGNATURE` is correct

**Request body**

Form fields (`Content-Type: application/x-www-form-urlencoded`), not JSON:

| Field | Type | Required | Rules |
|---|---|---|---|
| `MNT_TRANSACTION_ID` | string | yes | our payment UUID of the order (lowercase, with dashes); a number such as `15` is rejected |
| `MNT_OPERATION_ID` | string | yes | Moneta's id of the payment; stored as the order's payment id |
| `MNT_AMOUNT` | string | yes | rubles with two decimals; must equal the order total plus delivery |
| `MNT_ID` | string | yes | configured merchant account |
| `MNT_CURRENCY_CODE` | string | yes | RUB |
| `MNT_TEST_MODE` | string | yes | configured mode: 0 or 1 |
| `MNT_SUBSCRIBER_ID` | string | no | included in signature; omitted means empty |
| `MNT_SIGNATURE` | string | yes | checked with the shop secret |

```
MNT_ID=100&MNT_TRANSACTION_ID=15&MNT_OPERATION_ID=555&MNT_AMOUNT=790.00&MNT_CURRENCY_CODE=RUB&MNT_TEST_MODE=0&MNT_SIGNATURE=998b6268fe3a6c7f4eeb8ccec65a9566
```

Merchant ID, currency and test mode are signed as received and must match configuration. Subscriber ID is signed as received (empty when omitted). Duplicate MNT fields and malformed forms are rejected with 400. MNT_COMMAND must be absent on Pay URL.

**Response 200**

Plain text (`Content-Type: text/plain; charset=utf-8`), not JSON:

```
SUCCESS
```

**Errors**

The body is always the plain text `FAIL`. Moneta repeats the call later after any answer other than `SUCCESS`.

| Status | When |
|---|---|
| 400 | `MNT_TRANSACTION_ID` is not a canonical UUID, or `MNT_AMOUNT` differs from the order amount |
| 403 | the signature is wrong |
| 404 | the order does not exist |
| 500 | server error |

**Notes**

A valid callback sets the order to `paid`, submits it to iiko, and notifies the single staff chat once (order details followed by a separate phone message). The `confirmed` status remains readable for historical orders only. Repeated callbacks, and callbacks for an order that is already `paid`, `confirmed` or `canceled`, answer `SUCCESS` and change nothing. The signature is checked before the order is looked up, so an unsigned request gets 403 even for an unknown order. The route has no working-hours limit.

#### GET /webhooks/moneta

Same role, fields, responses and errors as POST /webhooks/moneta; fields are query parameters.

#### POST /webhooks/moneta/check

MONETA Check URL, public and authenticated by its signature. Accepts URL-encoded form data.

Required fields: MNT_COMMAND=CHECK, MNT_ID, MNT_TRANSACTION_ID (our payment UUID of the order; a numeric id or an unknown UUID gets result code 500), MNT_CURRENCY_CODE=RUB, MNT_TEST_MODE and MNT_SIGNATURE. Merchant and mode must match configuration. Optional signed fields: MNT_OPERATION_ID, MNT_AMOUNT (two decimals), MNT_SUBSCRIBER_ID. Missing optional fields contribute empty strings to the signature.

Example request (MONETA documentation test vector; account 54600817, integrity code QWERTY):

```text
MNT_COMMAND=CHECK&MNT_ID=54600817&MNT_TRANSACTION_ID=FF790ABCD&MNT_AMOUNT=120.25&MNT_CURRENCY_CODE=RUB&MNT_TEST_MODE=0&MNT_SIGNATURE=ea2d49048bdf11857f1b50270aedbc8d
```

This application's order IDs are positive integers, so the illustrative external ID above produces result code 500.

**Response 200:** signed UTF-8 XML, Content-Type application/xml; charset=utf-8:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<MNT_RESPONSE><MNT_ID>54600817</MNT_ID><MNT_TRANSACTION_ID>FF790ABCD</MNT_TRANSACTION_ID><MNT_RESULT_CODE>500</MNT_RESULT_CODE><MNT_SIGNATURE>373cc5df0d19d0e98eb4ebfceaa9cd38</MNT_SIGNATURE></MNT_RESPONSE>
```

Result codes: 100 = payable order with omitted request amount; 402 = payable order with matching amount; 200 = recorded payment; 500 = missing, canceled, nonpayable order or amount mismatch. Existing orders include MNT_AMOUNT as the stored total plus delivery. Canceled status takes precedence over a recorded payment. Unpaid orders must have been created today in APP_TIMEZONE (default Europe/Moscow) and pass current restaurant hours [start, end). Otherwise code 500 is returned, including for previously issued links. Recorded payments retain code 200 regardless of age/hours, except canceled orders. Valid delayed payment confirmations are still accepted; CHECK never confirms a payment.

Response signature is MD5(result code + merchant ID + transaction ID + integrity code). Check requests never update orders or send notifications.

**Errors:** 400 for malformed forms, duplicate MNT fields or a command other than CHECK; 403 for invalid signature or merchant/currency/mode mismatch; 500 for database errors, missing payment-hours configuration or hours lookup failures. Error bodies are plain text FAIL followed by a newline.

#### GET /webhooks/moneta/check

Same contract as POST /webhooks/moneta/check; fields are query parameters.

### Uploads

#### POST /uploads

Uploads an image and returns its URL. The admin panel uses it for photos of rooms, news and content blocks.

- **Role:** admin

**Request body**

`Content-Type: multipart/form-data`, not JSON.

| Field | Type | Required | Rules |
|---|---|---|---|
| `file` | file | yes | a JPEG, PNG or WebP image, at most 10 MiB; the type is detected from the content, the file name and the part's `Content-Type` are ignored |

```
POST /api/v1/uploads
Authorization: Bearer <token>
Content-Type: multipart/form-data; boundary=...

--...
Content-Disposition: form-data; name="file"; filename="room.png"
Content-Type: image/png

<bytes>
--...--
```

**Response 201**

```json
{ "url": "http://localhost:3000/uploads/2026/10/c19b550d7d3fc582866bd4e85f9301b7.png" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | the request has no `file` field, or is not a multipart form | `Выберите файл` |
| 400 | the file is not a JPEG, PNG or WebP image | `Допустимы только JPEG, PNG и WebP` |
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |
| 413 | the file is larger than 10 MiB | `Файл больше 10 МБ` |

**Notes**

The `url` is what you send as a photo URL to the content endpoints. The server also accepts it back without the origin (`/uploads/2026/10/...`), and accepts photos that ship with the site (`/images/room-vip.jpg`) and an empty string for "no photo"; any other address is rejected by the content endpoints. Files that are never used are not cleaned up. Uploaded files are static files served at `GET /uploads/...`, outside `/api/v1`; they are immutable and cached for a year, so a replaced photo is a new upload with a new URL.

#### PUT /categories/{id}/ordering-hours

Role: admin. Sets ordering hours and weekdays for one category. Settings survive iiko sync. Uses the existing restaurant-local clock for availability; start is inclusive and end exclusive. Overnight windows are not supported. Restaurant working hours still apply.

Path `id`: positive category ID. Required JSON fields:

| Field | Type | Rules |
|---|---|---|
| startTime | string | H:MM or HH:MM; empty together with endTime removes the time limit |
| endTime | string | Later than startTime, up to 24:00 |
| days | string | Seven 0/1 characters, Monday first; empty = every day; at least one day required |

Example: `{"startTime":"9:00","endTime":"14:00","days":"1111100"}`.

200 returns the updated Category object with `id`, `name`, `description`, `images`, `isAvailableForOrder`, `salesDays`, `startTime`, `endTime`, as in GET /categories/{id}. `salesDays` lists selected weekdays in Russian.

400: `Ошибка валидации, пожалуйста проверьте введенные вами данные`, with field errors: `Укажите время или пустую строку`, `Укажите время в формате ЧЧ:ММ`, `Время окончания должно быть позже начала`, or `Выберите дни недели`. Standard malformed JSON / invalid ID errors also apply. 401: `Требуется авторизация`. 403: `Недостаточно прав`. 404: `Не найдено` for missing or deleted categories.

### Promo codes

#### GET /promo-codes

- **Role:** admin.

Returns every campaign, including disabled campaigns, ordered by descending id. Empty result is `[]`.

**Request example:** `GET /api/v1/promo-codes` with `Authorization: Bearer <admin-token>`. No body.

**Response 200** (captured campaign from the admin CRUD test)

```json
[{"id": 1, "code": "TEST_10", "percentage": 10, "enabled": true, "target": "all", "productIds": [], "categoryIds": [], "minimumEligibleSubtotal": null, "maximumDiscount": null, "startsAt": null, "endsAt": null}]
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | Missing/invalid token | `Требуется авторизация` |
| 403 | Non-admin credential | `Недостаточно прав` |
| 500 | Unexpected database error | `Внутренняя ошибка сервера` |

#### GET /promo-codes/{id}

- **Role:** admin.

Returns the full campaign configuration.

**Path parameters:** id — positive integer campaign ID.

**Request example:** `GET /api/v1/promo-codes/1` with `Authorization: Bearer <admin-token>`. No body.

**Response 200** (captured campaign from the admin CRUD test)

```json
{"id": 1, "code": "TEST_10", "percentage": 10, "enabled": true, "target": "all", "productIds": [], "categoryIds": [], "minimumEligibleSubtotal": null, "maximumDiscount": null, "startsAt": null, "endsAt": null}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | Missing/invalid token | `Требуется авторизация` |
| 403 | Non-admin credential | `Недостаточно прав` |
| 500 | Unexpected database error | `Внутренняя ошибка сервера` |
| 404 | Missing campaign | `Промокод не найден.` |
| 404 | Invalid path ID | `Не найдено` |

#### POST /promo-codes

- **Role:** admin.

Returns the full campaign configuration.

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| code | string | yes | Trim, uppercase ASCII; `[A-Z0-9_-]{3,32}`; permanently unique and immutable |
| percentage | integer | yes | 1–99 |
| enabled | boolean | no | Defaults false when omitted |
| target | string | yes | all or selected |
| productIds / categoryIds | integer arrays | no | Existing IDs; sorted/deduplicated; union targeting. all requires empty arrays; selected requires at least one ID |
| minimumEligibleSubtotal / maximumDiscount | number or null | no | Positive rubles when set; omitted means null |
| startsAt / endsAt | RFC3339 string or null | no | End must follow start when both supplied; omitted means null |

```json
{"code": "TEST_10", "percentage": 10, "enabled": true, "target": "all", "productIds": [], "categoryIds": [], "minimumEligibleSubtotal": null, "maximumDiscount": null, "startsAt": null, "endsAt": null}
```

Writes are transactional. PUT fully replaces writable configuration; code cannot change.

**Response 201** (captured campaign from the admin CRUD test)

```json
{"id": 1, "code": "TEST_10", "percentage": 10, "enabled": true, "target": "all", "productIds": [], "categoryIds": [], "minimumEligibleSubtotal": null, "maximumDiscount": null, "startsAt": null, "endsAt": null}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | Missing/invalid token | `Требуется авторизация` |
| 403 | Non-admin credential | `Недостаточно прав` |
| 500 | Unexpected database error | `Внутренняя ошибка сервера` |
| 400 | Malformed JSON, wrong JSON types or invalid timestamp | `Некорректный JSON` |
| 400 | code | `Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.` |
| 400 | percentage | `Укажите целый процент от 1 до 99.` |
| 400 | minimumEligibleSubtotal / maximumDiscount | `Укажите положительную сумму.` |
| 400 | endsAt | `Дата окончания должна быть позже даты начала.` |
| 400 | target | `Выберите все блюда или укажите блюда и категории.` |
| 400 | productIds / categoryIds | `Выбранное блюдо или категория не найдены.` |
| 409 | Duplicate normalized code | `Такой промокод уже существует.` |

Validation uses message `Ошибка валидации, пожалуйста проверьте введенные вами данные` and the indicated key in `errors`. Wrong JSON types are malformed JSON errors.

#### PUT /promo-codes/{id}

- **Role:** admin.

Returns the full campaign configuration.

**Path parameters:** id — positive integer campaign ID.

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| code | string | yes | Trim, uppercase ASCII; `[A-Z0-9_-]{3,32}`; permanently unique and immutable |
| percentage | integer | yes | 1–99 |
| enabled | boolean | no | Defaults false when omitted |
| target | string | yes | all or selected |
| productIds / categoryIds | integer arrays | no | Existing IDs; sorted/deduplicated; union targeting. all requires empty arrays; selected requires at least one ID |
| minimumEligibleSubtotal / maximumDiscount | number or null | no | Positive rubles when set; omitted means null |
| startsAt / endsAt | RFC3339 string or null | no | End must follow start when both supplied; omitted means null |

```json
{"code": "TEST_10", "percentage": 10, "enabled": true, "target": "all", "productIds": [], "categoryIds": [], "minimumEligibleSubtotal": null, "maximumDiscount": null, "startsAt": null, "endsAt": null}
```

Writes are transactional. PUT fully replaces writable configuration; code cannot change.

**Response 200** (captured campaign from the admin CRUD test)

```json
{"id": 1, "code": "TEST_10", "percentage": 10, "enabled": true, "target": "all", "productIds": [], "categoryIds": [], "minimumEligibleSubtotal": null, "maximumDiscount": null, "startsAt": null, "endsAt": null}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | Missing/invalid token | `Требуется авторизация` |
| 403 | Non-admin credential | `Недостаточно прав` |
| 500 | Unexpected database error | `Внутренняя ошибка сервера` |
| 404 | Missing campaign | `Промокод не найден.` |
| 404 | Invalid path ID | `Не найдено` |
| 400 | Malformed JSON, wrong JSON types or invalid timestamp | `Некорректный JSON` |
| 400 | code | `Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.` |
| 400 | percentage | `Укажите целый процент от 1 до 99.` |
| 400 | minimumEligibleSubtotal / maximumDiscount | `Укажите положительную сумму.` |
| 400 | endsAt | `Дата окончания должна быть позже даты начала.` |
| 400 | target | `Выберите все блюда или укажите блюда и категории.` |
| 400 | productIds / categoryIds | `Выбранное блюдо или категория не найдены.` |
| 409 | Duplicate normalized code | `Такой промокод уже существует.` |

Validation uses message `Ошибка валидации, пожалуйста проверьте введенные вами данные` and the indicated key in `errors`. Wrong JSON types are malformed JSON errors.

Attempting to edit code returns 409: `Код нельзя изменить после создания.`

#### POST /orders/quote

Computes current authoritative pricing without creating orders, customers, addresses, reservations or payment links. Quotes do not reserve a price; availability, campaign configuration and delivery may change before checkout.

- **Role:** public, optional authentication. Invalid supplied credentials return 401.
- **Cache:** private, no-store, including errors. Working-hours middleware for order placement is not applied; product and category ordering windows and manual dish blocks are checked using the current clock.

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| items | array | yes | At least one item; each has positive integer productId and amount. Duplicate products are merged with checked quantity addition |
| fulfillment | string | yes | delivery or pickup |
| city | string | for delivery | Exact existing town name; ignored for pickup |
| promoCode | string | no | Trim outer whitespace, uppercase ASCII, validate `[A-Z0-9_-]{3,32}`. Empty/omitted means no discount |

```json
{"items": [{"productId": 1, "amount": 1}, {"productId": 2, "amount": 1}], "fulfillment": "delivery", "city": "Сочи", "promoCode": "SAVE10"}
```

**Response 200** (captured from TestQuote)

```json
{"subtotal": 699, "eligibleSubtotal": 699, "discount": 69.9, "total": 629.1, "deliveryPrice": 150, "payableTotal": 779.1, "promoCode": "SAVE10", "items": [{"productId": 1, "amount": 1, "offeredPrice": 599, "discount": 59.9}, {"productId": 2, "amount": 1, "offeredPrice": 100, "discount": 10}]}
```

Amounts are rubles. subtotal is original food, eligibleSubtotal is eligible food before discount, discount is the saved calculation, total is discounted food, deliveryPrice is separate, payableTotal is their sum. The example town has price 150 and freeFrom 650: food 699 qualified before discount, but discounted food 629.10 requires paid delivery.
Items retain original unit offeredPrice and merged quantity; discount is per line, never per unit. Largest-remainder allocation breaks ties by ascending productId. No-code quotes have eligibleSubtotal=0, discount=0, promoCode=null and zero line discounts. Pickup deliveryPrice=0. A zero discount is allowed when integer rounding yields zero; an empty/nonpositive eligible subtotal is rejected for a supplied code.

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | Malformed body/types | `Некорректный JSON` |
| 400 | Missing/invalid items, amount, fulfillment or required city | `Ошибка валидации, пожалуйста проверьте введенные вами данные` with standard field errors |
| 400 | Repeated amounts overflow | `Некорректное количество.` |
| 400 | Missing/deleted product | `Продукт не найден.` |
| 400 | Unknown delivery town | `Указано не существующее имя населенного пункта.` |
| 400 | Global promo switch disabled; errors.promoCode | `Промокоды временно отключены.` |
| 400 | Malformed promoCode | `Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.` |
| 400 | Unknown, disabled, future or expired promoCode | `Промокод недействителен.` |
| 400 | No eligible positive food spend | `В корзине нет блюд для этого промокода.` |
| 400 | Eligible subtotal below minimum | `Недостаточная сумма подходящих блюд для промокода.` |
| 401 | Invalid supplied credential | `Требуется авторизация` |
| 503 | Manually blocked dish | `Заказ блюда «{name}» временно недоступен.` |
| 503 | Category or product closed by day | `Возможность заказа "{name}" доступна только в следующие дни недели: {days}` |
| 503 | Category or product closed by time | `Время заказа "{name}" ограничено с {start} до {end}.` |
| 500 | Database errors or unrepresentable monetary totals | `Внутренняя ошибка сервера` |

The global settings switch defaults to false; no-code checkout and saved-order retries remain available when disabled. Promo failures use the standard validation message with the exact text above in errors.promoCode. Dates use start inclusive and end exclusive. Eligibility is the union of current category membership and explicit product selections; overlap never doubles a discount. Discount is floor(eligibleSubtotal * percentage / 100), limited by optional cap. Delivery thresholds use discounted food total.

### Delivery information

These fields are informational captions. Authoritative town tariffs and free-delivery thresholds remain in `/delivery-prices`; these settings never change order pricing. Text is plain text and must be rendered escaped. No additional cache policy is applied.

#### GET /delivery-settings

Role: **public**. No parameters or request body.

Request: `GET /api/v1/delivery-settings`.

**200** — current information:

```json
{"cooking":"~30 мин","outskirtsLabel":"СОТ и ДНТ","outskirtsMin":200,"outskirtsMax":350,"updatedAt":"2026-11-01T10:23:30.663667Z"}
```

Errors: **404** `Не найдено` (singleton missing); **500** `Внутренняя ошибка сервера`. GET never recreates a missing row.

#### PUT /delivery-settings

Role: **admin**, Bearer JWT. Atomically replaces all four fields; does not recreate a missing row.

| Field | Rules |
|---|---|
| `cooking` | Required plain text, trimmed, 1–100 Unicode characters |
| `outskirtsLabel` | Required plain text, trimmed, 1–60 Unicode characters |
| `outskirtsMin` | Required non-null integer rubles, 0–100000 |
| `outskirtsMax` | Required non-null integer rubles, 0–100000, at least `outskirtsMin` |

Zero and equal bounds are accepted. Fractional numbers, strings and booleans are rejected as malformed JSON. Invalid requests leave the saved row unchanged.

Request:

```json
{"cooking":"~30 мин","outskirtsLabel":"СОТ и ДНТ","outskirtsMin":200,"outskirtsMax":350}
```

**200** — saved fields with server-generated RFC 3339 `updatedAt`, in the same shape as GET above.

Errors:

- **400** `Некорректный JSON` for malformed JSON or wrong types.
- **400** `Ошибка валидации, пожалуйста проверьте введенные вами данные`, with field errors: `поле не должно быть пустым` for missing/null prices or empty text; `значение должно быть не больше 100` / `значение должно быть не больше 60` for overlong text; `значение должно быть не меньше 0` / `значение должно быть не больше 100000` for price bounds.
- **400** reversed range, example captured from tests:

```json
{"statusCode":400,"message":"Ошибка валидации, пожалуйста проверьте введенные вами данные","errors":{"outskirtsMax":"Максимальная цена должна быть не меньше минимальной"}}
```

- **401** `Требуется авторизация`.
- **403** `Недостаточно прав`.
- **404** `Не найдено`.
- **500** `Внутренняя ошибка сервера`.

### Homepage menu

#### GET /homepage-menu

Role: public. No parameters or request body. Returns **200** with the saved, ordered configuration:

```json
{"categories":[{"categoryId":2,"productIds":[]},{"categoryId":1,"productIds":[2,1]}]}
```

This is raw configuration: deleted, hidden or moved catalog IDs may remain until staff repair them.
Resolve it against the current visible menu. Empty `categories` means the first four visible categories;
empty `productIds` means up to three automatic suggestions from that category. Manual selections are not topped up.
Ordering blocks and closed ordering hours do not remove a selected dish; apply the existing ordering controls.
No valid manual products means automatic suggestions. Omit empty categories and hide the teaser when no categories remain.
The initial response is `{"categories":[]}`. iiko sync preserves this configuration.

Errors: **404** `Не найдено` if the configuration row is missing; **500** `Внутренняя ошибка сервера` for storage failures.

#### PUT /homepage-menu

Role: admin (Bearer JWT). Atomically replaces the entire configuration and returns **200** with the same shape.
Example request: `{"categories":[{"categoryId":2,"productIds":[]},{"categoryId":1,"productIds":[2,1]}]}`.
Both arrays are required and non-null. Arrays define display order; product IDs must belong to the specified category.
Unavailable ordering hours and manual ordering blocks are allowed. Hidden/deleted/missing categories and deleted/missing/moved products are rejected.

Errors:

| Status | Message / field errors |
|---|---|
| 400 | `Некорректный JSON` for malformed JSON or incorrect field types |
| 400 | `Ошибка валидации, пожалуйста проверьте введенные вами данные`; indexed keys below |
| 401 | `Требуется авторизация` |
| 403 | `Недостаточно прав` |
| 404 | `Не найдено` if the configuration row is missing |
| 500 | `Внутренняя ошибка сервера` |

Validation fields (first error returned; rejected saves leave configuration unchanged):

- `categories`: `Укажите список категорий` (missing/null).
- `categories[i].categoryId`: `Укажите уникальную категорию` (non-positive/duplicate) or `Категория недоступна` (missing/deleted/hidden).
- `categories[i].productIds`: `Укажите список из не более трёх блюд` (missing/null or more than three), `Укажите уникальные блюда` (non-positive/duplicate), or `Блюдо недоступно или относится к другой категории` (missing/deleted/moved).
