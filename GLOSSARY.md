# Glossary

Terms used across `citypark-go`, `citypark-modern` and `citypark-admin`. Decisions behind them: `docs/adr/`.

**Admin**: a user with role `admin` in `citypark-go`. The only role that can use `citypark-admin`.

**Content block**: one editable piece of page copy stored in `citypark-go` (`content_blocks` table), addressed by its
**block key**. Types: `text` (one plain line), `lines` (a plain list, one item per line), `richtext` (sanitized HTML,
*short* profile), `image` (photo with `alt` and focal point). Not used for lists with structure (rooms, news).

**Block key**: the dotted id of a content block, `<page>.<section>.<name>`, for example `home.hero.title`. Created only
by backend migrations; the admin edits values, never keys.

**Focal point**: the CSS `object-position` of a photo (`"50% 30%"`), set by clicking on the photo in the admin. The site
uses it when a photo is cropped. Stored as `position` next to the photo URL.

**Rich text profile**: the set of HTML tags a rich text field allows. *article* (news body): `p`, `h2`, `h3`, `strong`,
`em`, `a`, `ul`, `ol`, `li`, `blockquote`, `br`. *short* (room descriptions, hotel texts, home page blocks): the same
without `h2`, `h3` and `blockquote`. The admin toolbar and the Go sanitizer use the same list.

**Price placeholder**: `{{name}}` inside a content block, replaced by the site with a formatted price from the hotel
prices or the rooms (`{{minRoomPrice}}`, `{{lateCheckoutPerHour}}`, `{{smokingFine}}`, `{{breakfastDelivery}}`).
Keeps every price in one place.

**Hotel prices**: the hotel's fees (breakfast delivery, late checkout per hour, smoking fine) and the breakfast menu,
stored as numbers in `citypark-go`.

**Menu sync**: copying the iiko external menu into `categories` and `products`. Started by the timer, by the admin's
«Обновить меню из iiko» button or by the `X-Update-Token` endpoint; never two at once.

**Dish photo**: the product's button image in iiko, saved by the menu sync as a `product_images` row with
`type = 'from_iiko'`. Not edited in the admin.

**Hotel report**: data from the 1C database `CityParkHotel`, read through `citypark-go` (`/reports/...`).
