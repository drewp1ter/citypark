// Flow check: rooms. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/rooms.mjs   (server must run on BASE_URL, default http://localhost:3111)
// Room availability now comes from whole-stay booking quotes in the isolated 1C base.
// Obtain approval before running: searches persist advisory quotes. Today's room-count webhook does not set inventory.
import { chromium } from 'playwright'
const base = process.env.BASE_URL ?? 'http://localhost:3111'
const browser = await chromium.launch()
const out = (k, v) => console.log(k.padEnd(56), v)
const txt = async (loc) =>
	(
		await loc
			.first()
			.innerText()
			.catch(() => '—')
	).replace(/\s+/g, ' ')
const iso = (d) =>
	`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const plus = (n) => {
	const d = new Date()
	d.setDate(d.getDate() + n)
	return iso(d)
}

async function open(path, viewport = { width: 1440, height: 900 }) {
	const ctx = await browser.newContext({ viewport })
	const p = await ctx.newPage()
	const problems = []
	p.on('pageerror', (e) => problems.push(e.message.slice(0, 120)))
	const res = await p.goto(base + path)
	await p.waitForTimeout(500)
	return Object.assign(p, { problems, status: res.status() })
}

let p = await open('/rooms')
out(
	'default: status / h1 / cards',
	`${p.status} / ${await txt(p.locator('h1'))} / ${await p.locator('article').count()}`
)
out("chip 'Все · 5 типов' (spacing)", await txt(p.locator('[role=group] a').first()))
out(
	'VIP button (1 night default)',
	await txt(p.getByRole('link', { name: /Забронировать · |Проверить наличие и стоимость/ }))
)
out(
	'full-stay statuses (available/sold-out/unknown depend on current inventory and capacity)',
	JSON.stringify(await p.locator('article span.rounded-pill:visible').allInnerTexts())
)
out(
	'search defaults today / tomorrow',
	`${await p.locator('input[name=checkIn]').inputValue()} / ${await p.locator('input[name=checkOut]').inputValue()}`
)

// filters (links)
for (const [name, expected] of [
	['Одноместные', 2],
	['Двухместные', 2],
	['С балконом', 2],
]) {
	await p.getByRole('link', { name, exact: true }).click()
	await p.waitForTimeout(400)
	out(
		`filter ${name}: url / cards`,
		`${p.url().replace(base, '')} / ${await p.locator('article').count()} (want ${expected})`
	)
}
await p.getByRole('link', { name: 'Только свободные' }).click()
await p.waitForTimeout(400)
out(
	'+ only free (balcony; live inventory and guest capacity)',
	`${p.url().replace(base, '')} / ${await p.locator('article h2').allInnerTexts()}`
)
await p.getByRole('link', { name: 'Двухместные', exact: true }).click()
await p.waitForTimeout(400)
out('double + free (live inventory)', `${p.url().replace(base, '')} / ${await p.locator('article h2').allInnerTexts()}`)

// search keeps working and filters survive
await p.goto(base + '/rooms?type=single')
await p.locator('input[name=checkIn]').fill(plus(3))
await p.waitForTimeout(100)
const out3 = await p.locator('input[name=checkOut]').inputValue()
out('check-out follows check-in +3', `${out3} (want ${plus(4)})`)
await p.getByRole('button', { name: 'Найти' }).click()
await p.waitForURL('**/rooms?**')
await p.waitForTimeout(400)
out('search submit url', p.url().replace(base, ''))
out(
	'bar shows the searched dates',
	`${await p.locator('input[name=checkIn]').inputValue()} → ${await p.locator('input[name=checkOut]').inputValue()}, guests ${await p.locator('select[name=guests]').inputValue()}`
)

// nights in the VIP price
const d = await open(`/rooms?checkIn=${plus(2)}&checkOut=${plus(4)}&guests=4`)
out(
	'2 nights: VIP quote total (current admin rate)',
	await txt(d.getByRole('link', { name: /Забронировать · |Проверить наличие и стоимость/ }))
)
out('guests param -> select', await d.locator('select[name=guests]').inputValue())
// bad params are ignored
const bad = await open('/rooms?checkIn=2020-01-01&checkOut=nope&guests=99&type=hack&free=1')
out(
	'bad params: status / cards (current full-stay availability)',
	`${bad.status} / ${await bad.locator('article').count()}`
)
out(
	'bad params: dates fall back',
	`${await bad.locator('input[name=checkIn]').inputValue()} / ${await bad.locator('input[name=checkOut]').inputValue()}`
)
// Future stays use whole-stay inventory; outages/rate limits remain unknown.
const future = await open(`/rooms?checkIn=${plus(2)}&checkOut=${plus(3)}&free=1`)
out(
	'future + free: cards / availability badges (live inventory)',
	`${await future.locator('article').count()} / ${await future.locator('article span.rounded-pill:visible').count()}`
)
// nothing matches
const none = await open('/rooms?type=single&free=1')
await none.goto(base + '/rooms?type=double&free=1')
const empty = await open('/rooms?type=balcony&free=1')
out('balcony + free cards', String(await empty.locator('article').count()))
// make it empty: no room is both double and ... (euro is unavailable, twin is available) -> use an impossible mix via free+ type with none
// mobile
const m = await open('/rooms', { width: 390, height: 844 })
out(
	'mobile: cards / only-free chip hidden',
	`${await m.locator('article').count()} / ${await m.getByRole('link', { name: 'Только свободные' }).isVisible()}`
)
out('mobile: quick bar', await txt(m.locator("nav[aria-label='Быстрые действия']")))
out(
	'mobile: quick bar hrefs',
	JSON.stringify(
		await m.locator("nav[aria-label='Быстрые действия'] a").evaluateAll((as) => as.map((a) => a.getAttribute('href')))
	)
)
out('mobile: overflow', String(await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)))
out('mobile: amenities (short list)', await txt(m.locator('ul:visible').last()))
out('mobile: search button', await txt(m.getByRole('button', { name: /Найти/ })))
await m.getByRole('button', { name: /Найти/ }).click()
await m.waitForURL('**/rooms?**')
out('mobile submit keeps guests', m.url().replace(base, ''))
// hotel header phone + desktop phone link
out('header phone on /rooms', await p.getByRole('link', { name: 'Позвонить в отель' }).first().getAttribute('href'))
// tablet
const t = await open('/rooms', { width: 900, height: 900 })
out(
	'tablet 900: grid cols / overflow',
	`${await t.evaluate(() => getComputedStyle(document.querySelector('article').parentElement).gridTemplateColumns.split(' ').length)} / ${await t.evaluate(() => document.documentElement.scrollWidth > innerWidth)}`
)
// room pages
const link = await open('/rooms')
await link.getByRole('link', { name: 'VIP-номер', exact: true }).first().click()
await link.waitForURL('**/rooms/vip')
out('VIP name opens (want /rooms/vip)', link.url().replace(base, ''))

const vip = await open('/rooms/vip')
out('vip: status / h1 (want 200 / VIP-номер)', `${vip.status} / ${await txt(vip.locator('h1'))}`)
const thumbs = vip.locator('main section ul button')
out('vip: thumbnails (want 3)', String(await thumbs.count()))
out(
	'vip: description has «Камин электрический» (want true)',
	String((await vip.locator('main').innerText()).includes('Камин электрический'))
)
await thumbs.first().click()
const dialog = vip.locator('dialog[aria-label="Фотографии номера"]')
out('vip: dialog counter (want 2 из 4)', await txt(dialog.locator('p')))
await vip.keyboard.press('ArrowRight')
out('vip: after → (want 3 из 4)', await txt(dialog.locator('p')))
await vip.keyboard.press('Escape')
out('vip: dialog open after Esc (want false)', String(await dialog.evaluate((d) => d.open)))
out(
	'vip: focus back on the thumbnail (want true)',
	String(await thumbs.first().evaluate((b) => b === document.activeElement))
)
out('vip: page errors', JSON.stringify(vip.problems))

const single = await open('/rooms/single')
out('single: «Фото» section (want 0)', String(await single.getByRole('heading', { name: 'Фото' }).count()))
out(
	'single: description has «Сплит-система» (want true)',
	String((await single.locator('main').innerText()).includes('Сплит-система'))
)
out(
	'single: 3D-тур href (want /tour/room_1.html)',
	await single.getByRole('link', { name: /3D-тур/ }).getAttribute('href')
)
const twin = await open('/rooms/double-twin')
out('double-twin: description length (want > 0)', String((await twin.locator('main').innerText()).length > 200))
out(
	'double-twin: 3D-тур href (want /tour/room_2.html)',
	await twin.getByRole('link', { name: /3D-тур/ }).getAttribute('href')
)
const nope = await open('/rooms/nope')
out('nope: status (want 404)', String(nope.status))

out('page errors', JSON.stringify([...p.problems, ...m.problems, ...t.problems]))
await browser.close()
