# Changes from the NestJS backend

For whoever updates the frontend and the deployment. Task executors do not need this file.

## Routes

All routes moved from `/api` to `/api/v1`.

| Old | New | Notes |
|---|---|---|
| `GET /api` | `GET /health` | returns `{"status":"ok"}` |
| `POST /auth/sign-up` | same | still 201, also for duplicates |
| `POST /auth/sign-in` | same | 200 instead of 201; `roles: [...]` became `role: "user"`/`"admin"` |
| `GET /user`, `PUT /user` | same | `role` instead of `roles`; `PUT` validates and answers 409 on a taken email or phone |
| `POST /addresses` | same | body is `{city, address}`, no longer wrapped in `{address: {...}}` |
| `DELETE /addresses/:id` | same | 204 with no body |
| `POST /delivery-prices` | same | returns the created list; duplicate name is 409 (was 400, and other errors were swallowed) |
| `PATCH /delivery-prices/:id` | same | returns the object (was `true`/`false`); 404 when missing |
| `DELETE /delivery-prices/:id` | same | 204 |
| `GET /categories`, `GET /categories/:id` | same | deleted categories give 404; `isAviableForOrder` is now spelled `isAvailableForOrder`; no `createdAt`/`updatedAt` |
| `GET /categories/:id/products`, `GET /products` | same | same field rename; `categoryId` added; deleted products are excluded from search; `query` is optional |
| `POST /iiko/update-products` (token in body) and `POST /iiko/update-products/:token` | `POST /iiko/update-products` | token in the `X-Update-Token` header; disabled (404) when `IIKO_UPDATE_TOKEN` is empty |
| `POST /orders` | same | response adds `orderId`; an online order returns `amount` and `mntSignature` for the Moneta form instead of a payment URL; a bad amount is a validation error |
| `GET /orders/my`, `GET /orders`, `GET /orders/:id` | same | order shape changed, see `api.md`; admins can read any order |
| `POST /orders/:id` | `POST /orders/{id}/items` | returns the order |
| `DELETE /orders/:id/:productId` | `DELETE /orders/{id}/items/{productId}` | 204 |
| `DELETE /orders/:id` | same | 204 |
| `PATCH /orders/confirm/:id` | `POST /orders/{id}/confirm` | returns the order |
| `POST /webhook` (YooKassa) | `POST /webhooks/moneta` | the payment provider is now MONETA.RU: set the Pay URL in the Moneta cabinet; the callback is checked by its signature and amount and answered with `SUCCESS` / `FAIL` |
| `GET /docs` | `GET /api/v1/docs` | |

Error bodies are always `{statusCode, message, errors?}`. Validation `errors` keys are JSON paths (`items[0].amount`) instead of `property.constraint`.

Prices are still rubles in JSON. Order `total` is now a number, not a string, and orders carry `status` and `deliveryPrice`.

## Environment variables

| Old | New |
|---|---|
| `NODE_ENV` | `APP_ENV` |
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` | `DATABASE_URL` (PostgreSQL) |
| `APP_LOG_DIRECTORY` | removed; logs go to stdout |
| `THANK_YOU_PAGE_PYMENT_CASH` | `THANK_YOU_PAGE_PAYMENT_CASH` |
| `THANK_YOU_PAGE_PYMENT_ONLINE` | removed; the success page is set in the Moneta cabinet |
| `YOOKASSA_SHOP_ID`, `YOOKASSA_SECRET_KEY` | `MONETA_ID`, `MONETA_SECRET`, `MONETA_TEST_MODE` |
| `JWT_EXPIRES_IN=1d` | Go duration: `24h` |
| `TZ` (Docker image) | `APP_TIMEZONE`, default `Europe/Moscow` |
| — | `PORT`, `DEFAULT_PAGE_SIZE`, `IIKO_BASE_URL`, `TELEGRAM_BASE_URL` |

Unchanged: `DEBUG`, `JWT_SECRET`, `ADMIN_*`, `IIKO_API_LOGIN`, `IIKO_ORGANIZATION_ID`, `IIKO_CATEGORY_MARKER`, `IIKO_UPDATE_TOKEN`, `TELEGRAM_TOKEN`, `TELEGRAM_CHAT_ID_FOR_ORDER_CREATED`, `TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED`, `WORKING_HOURS_START`, `WORKING_HOURS_END`.

## Behaviour

- Passwords are bcrypt. Old accounts keep working: the legacy hash is checked on sign-in and replaced by bcrypt.
- Existing JWTs stop working (the payload changed from `roles` to `role`); users sign in again.
- Sale windows use `start <= now < end`. The old check had gaps around the first and last hour.
- A product's sale-window error now reports the product's own hours and days (it used to report the category's).
- The delivery price is stored on the order when it is created; later price changes do not affect old orders.
- The same product listed twice in one order is merged into one item.
- Tags are attached to a product by its own iiko tags only (the old code matched by substring across all products).
- Nutrition values are stored as decimals, not rounded to integers.
- Data moves with `cmd/migrate-legacy` (task 35) after the first iiko sync on the new database.
