# Task 67: Order contact and ownership safety

## Goal
Use immutable checkout contacts independently of authenticated ownership. Prevent
phone-based order disclosure and profile mutation before customer sessions launch.

## Depends on
66.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/store.go`,
`internal/order/service.go`, `internal/order/handler.go`.

## Files to create
- `migrations/00020_guest_checkout.sql`: nullable hashed guest checkout scope.
- `internal/auth/guest.go`, `internal/auth/guest_test.go`: purpose-specific signed guest cookies.
- `internal/order/ownership_test.go`: isolation and immutable snapshot regression tests.

## Files to change
- `internal/order/{types,store,service,handler}.go` and their tests: snapshots,
  ownership, optional authentication, duplicate isolation and private responses.
- `internal/auth/{middleware,middleware_test}.go`: strict optional authentication.
- `internal/user/{store,store_test}.go`: contact lookup without profile changes;
  exclude password hashes from JSON.
- `internal/seed/seed.go`, `internal/seed/seed_test.go`: explicit seed ownership.
- `docs/API.md`, `internal/apidocs/openapi.yaml`: describe changed semantics.
- `docs/tasks/PROGRESS.md`: register and complete this task.

## Spec
- `Order.AccountID *int64` is internal (`json:"-"`). History and detail access
  use only `account_id`, never phone or `customer_id`. Admin access is unchanged.
- Keep existing admin/customer response fields. Read contact/address text from
  snapshots, falling back to legacy references only for NULL snapshots. A deleted
  address retains snapshot text with `address.id = 0`; pickup remains null.
- `InsertParams` adds `AccountID *int64`, `GuestCheckoutHash string`,
  `ContactName *string`, `ContactPhone *string`. Insert atomically snapshots
  supplied contacts (or referenced user values for internal callers) and address
  text from the referenced address. Account ownership is always explicit.
- Checkout may find/create a contact by phone, but must never update an existing
  profile, email, hashes or role. It must not link the checkout address into any
  user's saved addresses, including when the contact phone belongs to an admin.
- `Tokens.Optional` allows no Authorization header; every present invalid,
  empty, repeated, expired or malformed header returns 401
  `Требуется авторизация`. Valid user/admin tokens set request identity.
  Verify the identity still exists and its current role matches before creating.
  Admin checkout creates an unowned order; staff Bearer routes stay unchanged.
- `Tokens.IssueGuestCheckout() (string, error)` issues a random, signed,
  purpose-specific JWT with 24-hour absolute expiry. `GuestCheckoutID(token)`
  verifies purpose, signature, expiry and random identifier. Guest proofs must
  not authenticate as users, nor may user tokens be used as guest proofs.
- Guest POST /orders uses a valid `citypark_guest_checkout` cookie or issues a
  new one. Cookie: HttpOnly, Secure, SameSite=Lax, Path=/api/v1/orders, MaxAge=86400.
  Store only SHA-256 of its random ID. Missing/invalid/expired cookie starts a
  fresh scope and never claims an old order. No raw guest IDs in JSON or logs.
- Internal `CreateInput.GuestCheckoutID` is excluded from JSON. Without a guest
  scope, direct service calls do not deduplicate. Ownership comes from context.
- `FindOpenDuplicate` requires `DuplicateScope{AccountID *int64,
  GuestCheckoutHash string}`. Match only the same account, or unowned orders with
  the same nonempty guest hash. Also match snapshot phone/name, delivery/pickup,
  address text, comment, delivery time and items hash; only new/pending statuses.
  No empty guest scope may match. Never deduplicate across accounts or guests.
- All order routes send `Cache-Control: private, no-store`, including errors.
  Password and legacy password hashes are excluded from User JSON.
- Adjust preexisting tests only where their old phone/profile/ownership semantics
  contradict this task. Preserve payment callbacks, notifications and iiko data.

Runtime methods retain their existing signatures except:

```go
func (t *Tokens) Optional(next http.Handler) http.Handler
func (s *Store) ListByAccount(ctx context.Context, accountID int64, limit, offset int) ([]Order, int, error)
func (s *Store) FindOpenDuplicate(ctx context.Context, scope DuplicateScope, name, phone string, addr *Address, comment, deliveryTime, itemsHash string) (id int64, found bool, err error)
```

Remove `ListByCustomer`; HTTP reads must have no contact-based ownership path.

| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| POST | /orders | public, optional Bearer | snapshot checkout and isolate retry scope | 201 |
| GET | /orders/my | user | paginated account-owned history | 200 |
| GET | /orders/{id} | user/admin | owned order, or any order for admin | 200 |
| GET | /orders | admin | search saved checkout contacts | 200 |

Request example (unchanged): `{"name":"Иван","phone":"+79991234567","items":[{"productId":1,"amount":1}]}`.
Response example (unchanged): `{"orderId":1,"redirectUrl":"https://pay.test/?x"}`.
The unchanged read response fields and other statuses/messages remain as defined
in `docs/API.md`; inaccessible guest/foreign details return 404 `Не найдено`.

New migration SQL:

```sql
ALTER TABLE orders ADD COLUMN guest_checkout_hash text;
CREATE INDEX orders_guest_checkout_idx ON orders (guest_checkout_hash, id DESC)
    WHERE account_id IS NULL AND guest_checkout_hash IS NOT NULL;
-- Down:
DROP INDEX orders_guest_checkout_idx;
ALTER TABLE orders DROP COLUMN guest_checkout_hash;
```

Snapshot insertion SQL (remaining item INSERT is unchanged):

```sql
INSERT INTO orders (customer_id, address_id, comment, delivery_time, items_hash,
    delivery_price_kopecks, account_id, guest_checkout_hash, contact_name,
    contact_phone, delivery_city, delivery_address)
VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''),
    COALESCE($9, (SELECT name FROM users WHERE id = $1)),
    COALESCE($10, (SELECT phone FROM users WHERE id = $1)),
    (SELECT city FROM addresses WHERE id = $2),
    (SELECT address FROM addresses WHERE id = $2)) RETURNING id;
```

Duplicate lookup SQL:

```sql
SELECT o.id FROM orders o WHERE o.status IN ('new', 'pending')
AND (($1::bigint IS NOT NULL AND o.account_id = $1)
 OR ($1::bigint IS NULL AND o.account_id IS NULL AND $2 <> ''
     AND o.guest_checkout_hash = $2))
AND o.contact_name = $3 AND o.contact_phone = $4 AND o.comment = $5
AND o.delivery_time = $6 AND o.items_hash = $7
AND o.delivery_city IS NOT DISTINCT FROM $8::text
AND o.delivery_address IS NOT DISTINCT FROM $9::text
ORDER BY o.id DESC LIMIT 1;
```

## Tests to write
- `TestOptional`: absent, empty, malformed, repeated, bad/expired, valid user/admin.
- `TestGuestCheckout`: random IDs, valid proof, tampering, expiry, wrong purpose,
  mutual rejection of guest and user tokens.
- `TestCheckoutContactSafety`: account/admin phone collision cannot change profile
  or credentials, link an address or claim an order; exact snapshots survive edits
  and address deletion; pending/paid notifications use snapshots.
- `TestLegacyOrderSnapshots`: NULL snapshots fall back to source records without acquiring ownership.
- `TestOrderOwnership`: shared phone/contact, distinct owner, guest, cross-account
  404, own paginated history; admin list/detail preserve their JSON keys.
- `TestDuplicateIsolation`: same scope retries, distinct account/guest separation,
  absent scope, same phone and items, contact/address/time changes, edited source.
- `TestCheckoutAuthentication`: invalid optional proof rejected before writes;
  valid user owns an order independent of contact; deleted/mismatched identity 401;
  admin creates unowned order. Assert no-store on successes and errors.
- `TestGuestCheckoutCookie`: first request issues secure cookie; retry reuses order;
  forged/expired/missing cookie starts fresh; cookie has no order-read authority.
- Read stored snapshots and scopes back using `testutil.NewDB`; use clock.Fixed,
  existing HTTP helpers and fake payments/notifier. No external service calls.

## API documentation
Update POST /orders, GET /orders/my, GET /orders/{id}, GET /orders notes and
OpenAPI descriptions/security/cookie response. No response shape changes or admin
type regeneration. Clients must forward the guest cookie for retry deduplication.

## Done when
All CONVENTIONS checks pass; changed internal packages (auth, order, user, seed)
have at least 80% statement coverage. Test against the configured database only.

## Out of scope
Provider identities, nullable account phone and verified email state (68),
customer sessions (69), SMTP/recovery/OAuth/Telegram/pairing (70–78), frontend,
production deployment and real payments.
