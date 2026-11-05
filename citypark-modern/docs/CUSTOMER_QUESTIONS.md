# Customer questions: source review and gaps

Reviewed for task 65 on 2026-11-05. This is a review of existing project facts, not a claim that the owner has newly approved every historical policy. No new policy is inferred from missing information.

## Published short answers

The /contacts#questions section answers these questions using existing public facts:

| Question | Source and treatment |
|---|---|
| Где находится «Сити Парк»? | Current GET /settings, through the same getSiteSettings result as the contact cards. No hardcoded address. |
| Есть ли парковка? | Exact existing copy from design/reference/light/desktop/contacts.html and ContactsPage: «Для гостей — бесплатная парковка у здания.». No claim about security, reservations or guaranteed spaces. |
| В какое время работают ресторан и доставка? | Current GET /working-hours, through the same getWorkingHours result as ContactsPage. Restaurant/delivery phone comes from settings. No promise that ordering immediately before closing guarantees arrival that day. |
| Где посмотреть номера, цены и правила проживания? | Links to /rooms and /hotel and the current hotel phone. No duplicate rules or fixed room prices. |
| Где посмотреть меню доставки и цены на блюда? | Link to /menu, which reads current backend products/categories. No price or delivery-time guarantee. |

These answers use existing card/link components and server rendering, remain visible without interaction and do not add FAQ structured data. Existing settings/hours fallbacks are retained, including their freshness limitations.

## Questions already answered elsewhere

| Customer question | Current source | Review finding |
|---|---|---|
| Когда можно заехать и выехать? | /hotel, hotel.rules.1.text and hotel.rules.check-in-mobile through getPageContent | Existing round-the-clock check-in, checkout and early check-in copy. Keep rules in the mutable source; link to the page. |
| Сколько стоит поздний выезд? | /hotel, hotel.rules.2.text plus getHotelPrices | Existing tiers and dynamic hourly fee. No new fixed price copied into FAQ. |
| Какие документы нужны, можно ли с детьми или животными? | /hotel rules and guest information from hotel content blocks | Existing policy copy is available. Do not rewrite or extend policy in this task. |
| Как оплатить проживание и забронировать номер? | /hotel hotel.rules.3.text and guest-info content | Existing cash/card, booking and payment copy. Some text distinguishes booking from payment on arrival; owner should review detailed policy before we summarize it further. |
| Сколько стоит доставка, когда она бесплатная? | /menu summary and /checkout town selection, backed by delivery-settings and delivery-prices | Location-dependent fees/thresholds already exist. Avoid a universal free-delivery claim; a future public conditions page should reuse the same data. |
| Сколько готовят заказ? | /menu summary from getDeliveryInfo | Existing approximate cooking estimate is not total delivery time. Do not turn it into an arrival guarantee. |
| Где посмотреть цены и фотографии номеров? | /rooms and /rooms/[id] through getRooms | Existing live content; linked from FAQ. |

Static hotel content defaults were inspected in src/entities/content/model/blocks.data.ts; deployed content may differ. Hotel/legal wording is left unchanged by this task.

## Owner facts still needed before publishing more answers

| Priority | Question to answer | What remains unknown |
|---|---|---|
| High | Есть ли минимальная сумма заказа и ограничения по районам? | Explicit minimum-order policy, complete service boundary and exceptions. Current fee/threshold data alone does not establish every restriction. |
| High | Сколько занимает доставка и когда принимается последний заказ? | Total delivery estimate, cutoff near closing and peak-load exceptions. Cooking estimate is insufficient. |
| High | Как отменить заказ и когда вернутся деньги? | Operational cancellation channel, stages at which cancellation is accepted and payment-provider refund timing. No promise published. |
| High | Как отменить бронь и действует ли предоплата? | Confirm the current booking/deposit/no-show/refund policy and consistency of short rules versus detailed guest/legal copy. |
| Medium | Парковка охраняется? Можно ли забронировать место? | Security, capacity, reservations, access restrictions and whether it covers restaurant visitors as well as hotel guests. Only the existing guest-parking statement is published. |
| Medium | Есть ли доступная среда и специальные номера? | Step-free access, lifts, bathrooms and accessible-room details; no inference from photos. |
| Medium | Можно ли заказать блюда при аллергии? | Verified allergen information and cross-contact handling. Dish descriptions are not an allergen-safety guarantee. |
| Medium | Как забронировать столик или банкет? | Confirm the preferred contact channel, booking conditions, capacity and deposit policy. |

Collect owner answers and their source/date in a later content task. Keep unresolved answers off the public site. Registration and indexing verification remain tasks 66–67.
