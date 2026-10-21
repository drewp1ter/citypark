# Pages and routes

Each page has 4 reference files and 4 screenshots:
`design/reference/{light|dark}/{desktop|mobile}/<page>.html` and
`design/screenshots/{light|dark}/{desktop|mobile}/<page>.jpg`.

Desktop designs are 1440 px wide (content container 1240 px). Mobile designs are 390 px wide.
Light and dark files have the same markup; only colours differ (see `tokens.css`).

| Route | Page file | Russian title | Main sections (top → bottom) |
|---|---|---|---|
| `/` | `home` | Главная | Header · Hero bento grid ("Отель. Ресторан. Доставка.", hotel card, restaurant card, delivery card, phone card) · 4 stat tiles · **Номера** (date bar + 3 room cards, VIP card is inverse) · **Ресторан** (terrace photo, banquet card in accent, 3D-panorama card, table booking form) · **Доставка** (category chips, 3 dish cards with "+", "Всё меню доставки" inverse card) · **Новости** (2 news cards + chef blog card) · Contacts block (address, 3 phones, route + e-mail buttons, map placeholder) · Footer wordmark |
| `/restaurant` | `restaurant` | Ресторан и летнее кафе | Hero · Ресторан / Летнее кафе / Банкеты blocks with photos · Table booking form |
| `/rooms` | `rooms` | Номера | Breadcrumbs · Search bar (Заезд, Выезд, Гости, Найти) · Filter chips (Все · 5 типов, Одноместные, Двухместные, С балконом, Только свободные) · 5 room cards (VIP, Одноместный с балконом, Одноместный, Двухместный 2 кровати, Двухместный «евро») · "В каждом номере" amenities · "Коротко" facts · "Удобнее по телефону?" CTA |
| `/hotel` | `hotel` | Отель-люкс | Hero · Номерной фонд · Балконы над парком · Бесплатно · Завтрак в номер · Правила (Заезд и выезд, Поздний выезд, Оплата, Гости, Дети до 14 лет, Нельзя) · Информация для гостей (documents) |
| `/menu` | `menu` | Доставка | Search · Category nav · "Постное меню" dish grid (dish cards with weight, price, +/−, size switch 500 г / 1 кг for pies) · **Cart panel** on desktop (sticky right column, "Корзина"); on mobile a sticky cart bar at the bottom |
| `/checkout` | `checkout` | Оформление заказа | Steps · **Получение** (Доставка / Самовывоз, Когда: В ближайшее время / Ко времени, Населённый пункт: Белореченск / СОТ-ДНТ, Улица и дом, Квартира, Подъезд/этаж) · **Контакты и оплата** (Имя, Телефон, Пожелания, payment: Картой курьеру / Наличными / СБП онлайн, privacy consent) · **Заказ** summary + "Подтвердить заказ" |
| `/news` | `news` | Новости | News grid · Блог шеф-повара (chef card + recipes) · "Акции — первыми в соцсетях" CTA |
| `/contacts` | `contacts` | Контакты | Contacts block (address, phones, e-mail, map) · "Напишите нам" form (Имя, Телефон или e-mail, Тема chips: Отель / Банкет / Доставка / Отзыв, Сообщение) · Реквизиты (company details) |

## Navigation

Desktop header (pill bar): logo "СИТИ ПАРК" · Ресторан · Отель · Номера · Доставка · Новости · Контакты ·
phone icon button · cart icon button · "Забронировать" (inverse pill).
Mobile header: logo · phone button · menu button (opens full-screen menu — **not designed**, build a simple
sheet with the same links using the existing tokens).

Map of design links → routes: `ModHome`→`/`, `ModRestaurant`→`/restaurant`, `ModRooms`→`/rooms`,
`ModHotel`→`/hotel`, `ModMenu`→`/menu`, `ModCheckout`→`/checkout`, `ModNews`→`/news`, `ModContacts`→`/contacts`.
In the reference HTML these are already rewritten to `<page>.html`.

## Not designed (build with existing components and tokens)

- Theme toggle (see CLAUDE.md).
- Mobile navigation sheet.
- Single news / blog post page (`/news/[slug]`): title, date, cover image, text — reuse NewsCard styles.
- Room detail page is optional; room cards can open a booking dialog or link to `/rooms#<id>`.
- 404 page: big Unbounded heading + link home.
- Empty cart state, form success/error states.
