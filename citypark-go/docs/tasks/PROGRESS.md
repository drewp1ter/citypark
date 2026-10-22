# Progress

Take the first unchecked task. Tick it only when every "Done when" command passes. The numbers in brackets are the tasks it depends on.

## Foundation
- [x] 01 Repo skeleton and health endpoint
- [x] 02 Config [01]
- [x] 03 httpx: JSON responses and errors [01]
- [x] 04 Logging and middleware [03]
- [x] 05 Validation [03]
- [x] 06 Store: database connection and transactions [01]
- [x] 07 Migrations and schema [06]
- [x] 08 Test utilities [07]
- [x] 09 Clock, sale windows and money [01]

## Auth and users
- [x] 10 Passwords and JWT [09]
- [x] 11 Auth middleware [03, 10]
- [x] 12 User store [08]
- [x] 13 Sign-up and sign-in [05, 10, 12]
- [x] 14 Current user endpoints and admin bootstrap [11, 13]

## Simple CRUD
- [x] 15 Addresses [08, 11]
- [x] 16 Delivery prices [08, 09, 11]

## Catalog
- [x] 17 Catalog store [08, 09]
- [x] 18 Category endpoints [17]
- [x] 19 Product endpoints [18]

## Integration clients
- [x] 20 iiko client [01]
- [x] 21 iiko sync: categories and products [08, 09, 20]
- [x] 22 iiko sync: images, tags, endpoint [21]
- [x] 23 Telegram client and order message [09]
- [x] 24 Moneta signatures [09]

## Orders
- [x] 25 Order store [12, 15, 17]
- [x] 25b Order store: payment state and duplicate lookup [25]
- [x] 26 Order service: create [16, 25b]
- [x] 27 Create-order endpoint and working hours [26]
- [x] 28 Order read endpoints [27]
- [x] 29 Order admin endpoints [28]
- [x] 30 Moneta payment callback [27]

## Shipping
- [x] 31 Application wiring [02, 04, 14, 16, 18, 19, 22, 23, 24, 27, 29, 30]
- [x] 32 Seed command [31]
- [x] 33 Dockerfile, env example, README [31]
- [x] 34 OpenAPI document [31]
- [-] 35 Legacy data migration command [31] SKIPPED

## iiko API v2 and menu/v3
- [x] 36 iiko v2 auth and config [33]
- [x] 37 iiko menu/v3 client and menus table [36]
- [x] 38 iiko sync from menu/v3 [37]
- [x] 39 Hidden categories [38]
- [x] 40 Product descriptions from iiko [39]
  - Not checked against real iiko: no credentials in this session. After `POST /iiko/update-products`, run `curl localhost:3000/api/v1/products?pageSize=500` and count non-empty `description`s.

## Hotel
- [x] 41 Hotel free rooms webhook [31]

## Admin panel support (see ../docs/adr/0001-admin-panel.md)
- [x] 42 Order search filters [29]
- [x] 43 Menu sync for admins, one sync at a time [38]
- [x] 44 Scheduled menu sync [43]
- [x] 45 Image uploads [31]
- [-] 46 Dish photos from the admin REMOVED: dish photos come from iiko (`syncImages`)
- [x] 47 Rich text sanitizer
  - Relative links get `rel="noreferrer"` only and external ones `rel="noreferrer noopener" target="_blank"`: bluemonday adds `noopener` only together with `target=_blank`. Tests pin this output.
- [ ] 48 Hotel rooms content [45, 47]
- [ ] 49 News [45, 47]
- [ ] 50 Content blocks [45, 47]
- [ ] 51 Site settings [31]
- [ ] 52 Hotel prices [41]
- [ ] 53 Hotel reports from 1C [31]
- [ ] 54 Content block rows for the home and hotel pages [50] (needs citypark-modern task 16)
