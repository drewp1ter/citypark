# API contract

Base path: `/api/v1`. JSON in, JSON out. This is the design reference; each task file repeats the part it implements.

Auth: `Authorization: Bearer <jwt>`. Role column: `-` public, `user` any signed-in user (admins included), `admin` admins only.

## Errors

```json
{ "statusCode": 400, "message": "Ошибка валидации, пожалуйста проверьте введенные вами данные", "errors": { "phone": "телефон должен быть действительным номером телефона" } }
```

| Status | Default message |
|---|---|
| 400 bad JSON | `Некорректный JSON` |
| 400 validation | `Ошибка валидации, пожалуйста проверьте введенные вами данные` |
| 401 | `Требуется авторизация` |
| 403 | `Недостаточно прав` |
| 404 | `Не найдено` |
| 500 | `Внутренняя ошибка сервера` |

## Routes

| Method | Path | Role | Request | Success |
|---|---|---|---|---|
| GET | `/health` | - | | 200 `{"status":"ok"}` |
| POST | `/auth/sign-up` | - | `{name, email?, phone, password}` | 201 `{}` (also when the email or phone is already taken) |
| POST | `/auth/sign-in` | - | `{email, password}` | 200 `User` + `token` |
| GET | `/user` | user | | 200 `User` |
| PUT | `/user` | user | `{name, email?, phone?}` | 200 `User` |
| GET | `/addresses` | user | | 200 `[Address]` |
| POST | `/addresses` | user | `{city, address}` | 201 `Address` |
| DELETE | `/addresses/{id}` | user | | 204 |
| GET | `/delivery-prices` | - | | 200 `[DeliveryPrice]` |
| POST | `/delivery-prices` | admin | `{deliveryPrices: [{name, value?, price}]}` | 201 `[DeliveryPrice]` |
| PATCH | `/delivery-prices/{id}` | admin | `{name, value?, price}` | 200 `DeliveryPrice` |
| DELETE | `/delivery-prices/{id}` | admin | | 204 |
| GET | `/categories` | - | | 200 `[Category]` |
| GET | `/categories/{id}` | - | | 200 `Category` |
| GET | `/categories/{id}/products` | - | | 200 `[Product]` |
| GET | `/products?query=&page=&pageSize=` | - | | 200 `[Product]` |
| POST | `/iiko/update-products` | header `X-Update-Token` | | 200 `{totalCategories, totalProducts}` |
| POST | `/orders` | - | `CreateOrder` | 201 `{orderId, redirectUrl}` (cash) or `{orderId, amount, mntSignature}` (online) |
| GET | `/orders/my?page=&pageSize=` | user | | 200 `{orders: [Order], count}` |
| GET | `/orders/{id}` | user | | 200 `Order` (owner or admin) |
| GET | `/orders?page=&pageSize=` | admin | | 200 `{orders: [Order], count}` |
| POST | `/orders/{id}/items` | admin | `{productId, amount}` | 200 `Order` |
| DELETE | `/orders/{id}/items/{productId}` | admin | | 204 |
| DELETE | `/orders/{id}` | admin | | 204 |
| POST | `/orders/{id}/confirm` | admin | | 200 `Order` |
| POST | `/webhooks/moneta` | - | Moneta payment callback (form fields `MNT_*`) | 200 text `SUCCESS`; otherwise text `FAIL` |

## Objects

```jsonc
// User
{ "id": 1, "name": "Иван", "email": "", "phone": "+79990000000", "role": "user" }

// Address
{ "id": 1, "city": "Сочи", "address": "ул. Ленина, 1", "createdAt": "...", "updatedAt": "..." }

// DeliveryPrice
{ "id": 1, "name": "Сочи", "value": "Россия, Краснодарский край, Сочи", "price": 150, "createdAt": "...", "updatedAt": "..." }

// Category
{ "id": 1, "name": "Мангал", "description": "", "images": [{ "id": 1, "url": "https://...", "type": "from_iiko" }],
  "isAvailableForOrder": true, "salesDays": "Пн, Вт, Ср, Чт, Пт", "startTime": "9:00", "endTime": "21:00" }

// Product
{ "id": 1, "categoryId": 1, "name": "Шашлык", "description": "", "measureUnit": "порц", "weight": 0.23,
  "proteins": 0, "fats": 0, "carbohydrates": 0, "calories": 0, "flags": 0, "price": 320,
  "images": [], "isAvailableForOrder": true, "salesDays": "", "startTime": "", "endTime": "" }

// CreateOrder
{ "phone": "+79991234567", "name": "Иван", "comment": "", "paymentType": "cash",
  "address": { "city": "Сочи", "address": "ул. Ленина, 1" },      // optional: no address means pickup
  "items": [{ "productId": 1, "amount": 2 }] }

// Order
{ "id": 1, "createdAt": "...", "updatedAt": "...", "status": "new", "comment": "", "paymentType": "cash",
  "total": 640, "deliveryPrice": 150,
  "customer": { "id": 2, "name": "Иван", "phone": "+79991234567" },
  "address": { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1" },   // or null
  "items": [{ "productId": 1, "name": "Шашлык", "categoryName": "Мангал", "amount": 2, "offeredPrice": 320 }] }
```

`total` is the sum of `offeredPrice * amount` and does not include `deliveryPrice`.

Order statuses: `new` → `pending` (the payment form was signed for an online order) → `paid` (Moneta confirmed the payment); an admin can set `confirmed`. `canceled` exists only for orders migrated from the old system.

Online payment: the frontend opens the Moneta payment form with `MNT_TRANSACTION_ID = orderId`, `MNT_AMOUNT = amount` and `MNT_SIGNATURE = mntSignature`. `amount` is a string with two decimals and includes the delivery price.

## Order creation errors

| Status | Message |
|---|---|
| 503 | `Извините, но мы не можем принять ваш заказ.\nЗаказы принимаются с {start} до {end}` |
| 503 | `Время заказа "{name}" ограничено с {start} до {end}.` |
| 503 | `Возможность заказа "{name}" доступна только в следующие дни недели: {days}` |
| 400 | `Продукт не найден.` |
| 400 | `Указано не существующее имя населенного пункта.` |
