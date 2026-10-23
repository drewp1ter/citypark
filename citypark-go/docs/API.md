# Citypark API — frontend reference

This file describes the endpoints that are implemented. Every task that adds or changes an endpoint updates it in the same commit (see `docs/tasks/CONVENTIONS.md`, "API documentation").

## General

- Base path: `/api/v1`. Requests and responses are JSON (`Content-Type: application/json`).
- Auth: `Authorization: Bearer <token>`. The token comes from `POST /auth/sign-in`.
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

Sections are grouped by area, in this order: Health, Auth, User, Addresses, Delivery prices, Catalog, Hotel, Rooms, News, Content, Settings, Orders, Reports, Integrations, Uploads. Add an area heading (`### Auth`) when its first endpoint is documented. Each endpoint uses this format:

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
| `phone` | string | yes | a Russian phone number in any common format; stored as `+7XXXXXXXXXX` |
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

The response is always `201 {}`, also when the email or phone is already registered. The server does not reveal whether an account exists, so no token is returned: call `POST /auth/sign-in` afterwards.

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

**Notes**

Send the token as `Authorization: Bearer <token>` on later requests. Unknown email and wrong password give the same 401.

### User

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

`email` is an empty string when the user has no email.

#### PUT /user

Updates the signed-in user's name, email and phone.

- **Role:** user

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `name` | string | yes | at most 80 characters |
| `email` | string | no | a valid email address; stored in lower case; empty removes the email |
| `phone` | string | no | a Russian phone number in any common format; stored as `+7XXXXXXXXXX`; empty keeps the current phone |

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

**Notes**

The request replaces the profile: omitting `email` or sending `""` removes the stored email, while omitting `phone` keeps the current one.

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

The answer is 204 also when the address does not exist or is not in the user's list. Only the user's link is removed; other users keep the address.

### Delivery prices

#### GET /delivery-prices

Returns the delivery price of every town, ordered by `id`.

- **Role:** public

**Response 200**

```json
[
  { "id": 1, "name": "Сочи", "value": "Россия, Краснодарский край, Сочи", "price": 150, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" }
]
```

**Notes**

The list is `[]` when there are no delivery prices. An order is matched to a price by comparing the address `city` with `name`.

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

```json
{ "deliveryPrices": [{ "name": "Сочи", "value": "Россия, Краснодарский край, Сочи", "price": 150 }, { "name": "Адлер", "price": 99.5 }] }
```

**Response 201**

```json
[
  { "id": 1, "name": "Сочи", "value": "Россия, Краснодарский край, Сочи", "price": 150, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" },
  { "id": 2, "name": "Адлер", "value": "", "price": 99.5, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:01:55.36535Z" }
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

Replaces the name, description and price of a delivery price.

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

```json
{ "name": "Сочи", "value": "Россия, Краснодарский край, Сочи", "price": 200 }
```

**Response 200**

```json
{ "id": 1, "name": "Сочи", "value": "Россия, Краснодарский край, Сочи", "price": 200, "createdAt": "2026-10-13T17:01:55.36535Z", "updatedAt": "2026-10-13T17:05:10.1182Z" }
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

An unknown or deleted category gives `200` with `[]`, not `404`. `price` is in rubles. `isAvailableForOrder` is `true` only when both the product's own sale window and its category's window are open now; `salesDays`, `startTime` and `endTime` describe the product's own window and are empty strings when it has none.

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

Returns the contact details.

- **Role:** public

**Response 200**

```json
{
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

### Orders

#### POST /orders

Places an order. The customer is found by phone number or created. Orders are accepted only during the restaurant's working hours.

- **Role:** public

**Request body**

| Field | Type | Required | Rules |
|---|---|---|---|
| `phone` | string | yes | a valid Russian phone number; it is stored as `+7XXXXXXXXXX` |
| `name` | string | yes | at most 80 characters |
| `comment` | string | no | at most 8192 characters |
| `paymentType` | string | yes | `cash` or `online` |
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
  "paymentType": "cash",
  "address": { "city": "Сочи", "address": "ул. Ленина, 1" },
  "items": [{ "productId": 1, "amount": 2 }]
}
```

**Response 201, `paymentType: "cash"`**

```json
{ "orderId": 1, "redirectUrl": "http://localhost:3001/thankyou" }
```

**Response 201, `paymentType: "online"`**

```json
{ "orderId": 1, "amount": "790.00", "mntSignature": "8c65fcd4afe3dc4cb915e04aaa273574" }
```

**Errors**

| Status | When | Message |
|---|---|---|
| 400 | the body is not valid JSON | `Некорректный JSON` |
| 400 | a field is missing or invalid; `errors` lists the fields | `Ошибка валидации, пожалуйста проверьте введенные вами данные` |
| 400 | the phone number cannot be normalized | `Неверный номер телефона.` |
| 400 | `address.city` is not a known delivery city | `Указано не существующее имя населенного пункта.` |
| 400 | a product does not exist or is deleted | `Продукт не найден.` |
| 503 | the current time is outside the working hours | `Извините, но мы не можем принять ваш заказ.` followed by a newline and `Заказы принимаются с 9:00 до 22:00` (the configured hours) |
| 503 | the product's or its category's weekday limit excludes today | `Возможность заказа "<name>" доступна только в следующие дни недели: Пн, Вт, Ср, Чт, Пт` |
| 503 | the product's or its category's time limit excludes now | `Время заказа "<name>" ограничено с 9:00 до 21:00.` |

**Notes**

For a cash order the frontend sends the customer to `redirectUrl`. For an online order the frontend opens the Moneta payment form with `orderId` as `MNT_TRANSACTION_ID`, `amount` as `MNT_AMOUNT` and `mntSignature` as `MNT_SIGNATURE`. Repeating an identical online order (same phone, address, comment and items) while the previous one is still unpaid returns the same `orderId`. The message of the working-hours error contains a real newline (`\n` in JSON); show it as two lines. The same product listed twice is merged into one item.

#### GET /orders/my

Lists the signed-in user's orders, newest first.

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
      "paymentType": "cash",
      "total": 599,
      "deliveryPrice": 150,
      "customer": { "id": 2, "name": "Иван", "phone": "+79991234567" },
      "address": { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1" },
      "items": [{ "productId": 1, "name": "Шашлык", "categoryName": "Мангал", "amount": 1, "offeredPrice": 599 }]
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

`count` is the total number of the user's orders, not the size of the page. `orders` is `[]` when there are none. `address` is `null` for a pickup order. `total` is the sum of the items without delivery; add `deliveryPrice` for the amount to pay.

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
  "paymentType": "cash",
  "total": 599,
  "deliveryPrice": 150,
  "customer": { "id": 2, "name": "Иван", "phone": "+79991234567" },
  "address": { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1" },
  "items": [{ "productId": 1, "name": "Шашлык", "categoryName": "Мангал", "amount": 1, "offeredPrice": 599 }]
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 404 | the order does not exist, is not a valid id, or belongs to another user | `Не найдено` |

**Notes**

An admin can read any order. For a normal user someone else's order looks the same as a missing one.

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
| `paymentType` | string | no | `cash` or `online` |

Example: `GET /orders?q=%D0%B8%D0%B2%D0%B0%D0%BD&from=2026-10-21&to=2026-10-21&status=paid&paymentType=online`.

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
      "paymentType": "cash",
      "total": 599,
      "deliveryPrice": 150,
      "customer": { "id": 2, "name": "Иван", "phone": "+79991234567" },
      "address": { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1" },
      "items": [{ "productId": 1, "name": "Шашлык", "categoryName": "Мангал", "amount": 1, "offeredPrice": 599 }]
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
| `status`, `paymentType` | `недопустимое значение` |

```json
{
  "statusCode": 400,
  "message": "Некорректные параметры поиска",
  "errors": {
    "from": "дата в формате ГГГГ-ММ-ДД",
    "paymentType": "недопустимое значение",
    "status": "недопустимое значение"
  }
}
```

**Notes**

`count` counts the matching orders, not the size of the page. `customer` shows who placed each order.

`q` matches the order number, any part of the customer's phone and the customer's name (case-insensitive). `%` and `_` in `q` are matched literally.

`from` and `to` are local dates (`APP_TIMEZONE`), both inclusive. `from` or `to` alone is allowed. An empty parameter (`?status=`) is the same as a missing one.

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

The updated order, in the same format as `GET /orders/{id}`.

```json
{
  "id": 2,
  "createdAt": "2024-05-24T12:00:00Z",
  "updatedAt": "2024-05-24T12:00:00Z",
  "status": "new",
  "comment": "",
  "paymentType": "cash",
  "total": 799,
  "deliveryPrice": 150,
  "customer": { "id": 2, "name": "Иван", "phone": "+79991234567" },
  "address": { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1" },
  "items": [
    { "productId": 1, "name": "Шашлык", "categoryName": "Мангал", "amount": 1, "offeredPrice": 599 },
    { "productId": 2, "name": "Лаваш", "categoryName": "Мангал", "amount": 2, "offeredPrice": 100 }
  ]
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

#### POST /orders/{id}/confirm

Sets the order status to `confirmed` and notifies the staff.

- **Role:** admin

**Parameters**

| Field | Type | Required | Rules |
|---|---|---|---|
| `id` | integer | yes | path parameter, the order id |

**Response 200**

The updated order, in the same format as `GET /orders/{id}`, with `"status": "confirmed"`.

```json
{
  "id": 2,
  "createdAt": "2024-05-24T12:00:00Z",
  "updatedAt": "2024-05-24T12:00:00Z",
  "status": "confirmed",
  "comment": "",
  "paymentType": "cash",
  "total": 799,
  "deliveryPrice": 150,
  "customer": { "id": 2, "name": "Иван", "phone": "+79991234567" },
  "address": { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1" },
  "items": [
    { "productId": 1, "name": "Шашлык", "categoryName": "Мангал", "amount": 1, "offeredPrice": 599 },
    { "productId": 2, "name": "Лаваш", "categoryName": "Мангал", "amount": 2, "offeredPrice": 100 }
  ]
}
```

**Errors**

| Status | When | Message |
|---|---|---|
| 401 | no token or invalid token | `Требуется авторизация` |
| 403 | the token belongs to a user who is not an admin | `Недостаточно прав` |
| 404 | the order does not exist or `id` is not a valid id | `Не найдено` |

**Notes**

The status is set whatever it was before, and every call notifies the staff again.

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

Products take their `description` from the iiko menu item (trimmed; an empty string when iiko has none, and an empty iiko description clears the old one on the next update). The update saves each product's iiko button image as a product image of type `from_iiko` (the first size with an image wins; an image url used twice keeps the last product), and deletes `from_iiko` images that are no longer in the menu. Categories get no images from iiko, so their `from_iiko` images are all deleted. The update does not take tags, nutrition values or sale windows from iiko: nutrition values and sale windows are reset and all tags are removed. Category descriptions are reset to an empty string. Images added by hand stay.

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
| `MNT_TRANSACTION_ID` | string | yes | our order id, a whole number from 1 |
| `MNT_OPERATION_ID` | string | yes | Moneta's id of the payment; stored as the order's payment id |
| `MNT_AMOUNT` | string | yes | rubles with two decimals; must equal the order total plus delivery |
| `MNT_SIGNATURE` | string | yes | checked with the shop secret |

```
MNT_ID=100&MNT_TRANSACTION_ID=15&MNT_OPERATION_ID=555&MNT_AMOUNT=790.00&MNT_CURRENCY_CODE=RUB&MNT_TEST_MODE=0&MNT_SIGNATURE=998b6268fe3a6c7f4eeb8ccec65a9566
```

The other `MNT_*` fields are ignored: the shop id, the currency and the test mode come from the server configuration.

**Response 200**

Plain text (`Content-Type: text/plain; charset=utf-8`), not JSON:

```
SUCCESS
```

**Errors**

The body is always the plain text `FAIL`. Moneta repeats the call later after any answer other than `SUCCESS`.

| Status | When |
|---|---|
| 400 | `MNT_TRANSACTION_ID` is not a number from 1, or `MNT_AMOUNT` differs from the order amount |
| 403 | the signature is wrong |
| 404 | the order does not exist |
| 500 | server error |

**Notes**

A valid callback sets the order to `paid` and notifies the staff once. Repeated callbacks, and callbacks for an order that is already `paid`, `confirmed` or `canceled`, answer `SUCCESS` and change nothing. The signature is checked before the order is looked up, so an unsigned request gets 403 even for an unknown order. The route has no working-hours limit.

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
