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

Sections are grouped by area, in this order: Health, Auth, User, Addresses, Delivery prices, Catalog, Hotel, Orders, Integrations. Add an area heading (`### Auth`) when its first endpoint is documented. Each endpoint uses this format:

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
| 500 | iiko could not be reached or the update failed; nothing is changed | `Внутренняя ошибка сервера` |

**Notes**

The update runs in one transaction and can take a few seconds. The menu is the external menu `IIKO_EXTERNAL_MENU_ID`. Every menu group becomes a category, even a group with nothing for sale. Items without a price are skipped. `totalCategories` and `totalProducts` count the categories and products that are in the menu after the update. Items that disappeared from iiko are hidden, not removed. The `isHidden` flag of a menu group is stored; hidden groups are synced with their products but are not listed by `GET /categories`.

Products take their `description` from the iiko menu item (trimmed; an empty string when iiko has none, and an empty iiko description clears the old one on the next update). The update no longer takes images, tags, nutrition values or sale windows from iiko, and it clears the ones that came from iiko before: nutrition values and sale windows are reset, `from_iiko` images are deleted and all tags are removed. Category descriptions are still reset to an empty string. Images added by hand stay.

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
