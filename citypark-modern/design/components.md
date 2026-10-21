# Component inventory

Build these as React components (Server Components by default; add `"use client"` only where noted).
All colours come from CSS variables in `tokens.css` — never hard-code hex values in components.
Find each component in the reference HTML by the page and section named in "Seen on".

## Layout

| Component | Seen on | Notes |
|---|---|---|
| `SiteHeader` | all | Desktop: white pill bar (`--surface`, `--border`, radius pill), logo, 6 nav links, phone icon button, cart icon button with count, "Забронировать" inverse pill, + **ThemeToggle** (new). Mobile: logo, phone button, menu button. Sticky on mobile. Client: cart count, menu open state. |
| `MobileNav` | — (not designed) | Full-screen sheet with the same links, phones and ThemeToggle. Client. |
| `SiteFooter` | all | Giant "СИТИ ПАРК" wordmark (Unbounded, `--text-wordmark`), © line, legal links (Документы, Оферта, Конфиденциальность, VK, OK). |
| `ContactsBlock` | home, contacts | Inverse block (`--inverse-bg`): address in Unbounded, rows with label + phone (`--inverse-border` dividers), "Построить маршрут" accent pill, e-mail outline pill, map area (`--inverse-surface`). Map: embed Yandex Maps later; keep the placeholder for now. |
| `Section` | all | `max-width: var(--container)`, side padding `--gutter-*`, title row (h2 in Unbounded + optional right-side action like "Все новости ↗" or chips). |
| `Breadcrumbs` | rooms | Small muted links. |

## Primitives

| Component | Notes |
|---|---|
| `Button` | Variants: `accent` (lime bg, `--on-accent` text), `inverse` (`--inverse-bg`, white text), `outline` (`--border-strong`), `ghost`. Sizes: md (44px), lg (52–56px). Always pill radius. |
| `IconButton` | 44×44 circle; variants as Button. Used for phone, cart, "+", "↗" arrow on cards. |
| `Chip` / `ChipGroup` | Pill filters and category tabs; selected = `--inverse-bg` + white text; others outline. Client when interactive. |
| `Badge` | Small accent pill with dot (e.g. "Белореченск · ул. Гоголя, 61"). |
| `Card` | `--surface`, 1px `--border`, radius `--radius-lg`/`--radius-xl`, overflow hidden. Variant `inverse` and `accent`. |
| `StatTile` | Big number (Unbounded) + muted caption. Home: 12 номеров, 24/7, 0 ₽, 30 мин. |
| `Field`, `Input`, `Select`, `Textarea`, `DateInput`, `PhoneInput`, `RadioCards`, `Checkbox` | Inputs are `--surface`/`--placeholder` filled, radius `--radius-xs`, min-height 48px, label above in `--text-small`. Phone mask `+7 ___ ___-__-__`. |
| `ThemeToggle` | **New.** Sun/moon icon button in the header (desktop: before the phone button; mobile: in MobileNav and header). Cycles light ↔ dark; first visit follows the OS. Client. |

## Domain components

| Component | Seen on | Notes |
|---|---|---|
| `HomeHero` | home | Bento grid: big intro card, hotel photo card (overlay label + arrow), restaurant card, delivery card, inverse phone card. |
| `RoomCard` | home, rooms | Photo, name, area/beds line, price "от 4 000 ₽", "Выбрать" button. `featured` variant (VIP) uses inverse colours and an accent button. |
| `StayDatesBar` | home, rooms | Заезд / Выезд date inputs (+ Гости on rooms) and "Проверить"/"Найти" accent button. Client. |
| `RoomFilters` | rooms | ChipGroup + "Только свободные" toggle. Client. |
| `AmenityList` | rooms, hotel | Icon + text grid ("В каждом номере", "Бесплатно"). |
| `RulesList` | hotel | Label/value rows separated by `--divider` (Заезд и выезд, Оплата, Дети до 14 лет…). |
| `DocumentList` | hotel | "Информация для гостей" links to PDFs. |
| `TableBookingForm` | home, restaurant | Дата, Время, Гостей chips (2 / 3–4 / 5–8 / Банкет), Телефон, "Отправить заявку". Client + server action. |
| `PromoCard` | home, restaurant | Accent card ("Банкеты на 50–60 гостей") and plain card ("3D-панорама зала"). |
| `DishCard` | home, menu | Photo, name, weight · price, "+" accent IconButton; in cart becomes − qty + stepper. Optional size switch (500 г / 1 кг). Client (cart). |
| `MenuCategoryNav` | home, menu | Horizontal chips; on menu page also a search field. |
| `CartPanel` | menu (desktop) | Sticky right column: lines with steppers, totals, delivery note, "Оформить" button → `/checkout`. Client. |
| `CartBar` | menu, mobile | Sticky bottom bar: count badge, "Корзина", total → `/checkout`. Client. |
| `CheckoutForm` | checkout | Three blocks (Получение, Контакты и оплата, Заказ summary), "Подтвердить заказ". Client + server action. Validate phone and address. |
| `NewsCard` | home, news | Image, date, title. |
| `ChefBlogCard` | home, news | Chef avatar, "Блог шефа · Денис Троицкий", recipe title, "Читать ↗". |
| `SocialCTA` | news | "Акции — первыми в соцсетях" with VK / OK buttons. |
| `ContactForm` | contacts | Имя, Телефон или e-mail, Тема chips, Сообщение, "Отправить". Client + server action. |
| `Requisites` | contacts | Company details table. |
