// Flow check: checkout-and-contacts. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/checkout-and-contacts.mjs   (server must run on BASE_URL, default http://localhost:3111)
import { chromium } from 'playwright'
import { seedCart } from '../menu-cart.mjs'
const base = process.env.BASE_URL ?? 'http://localhost:3111'
const browser = await chromium.launch()
const out = (k, v) => console.log(k.padEnd(58), v)
const CART = await seedCart([1, 2, 1])
async function pageWithCart(cart = CART, viewport = { width: 1440, height: 1000 }) {
	const ctx = await browser.newContext({ viewport })
	if (cart) await ctx.addInitScript((c) => localStorage.setItem('citypark-cart', JSON.stringify(c)), cart)
	return ctx.newPage()
}
const txt = async (loc) =>
	(
		await loc
			.first()
			.innerText()
			.catch(() => '—')
	).replace(/\s+/g, ' ')

// ---------- checkout
let p = await pageWithCart()
await p.goto(base + '/checkout')
await p.waitForTimeout(500)
out('summary: lines + total', await txt(p.locator("aside[aria-label='Ваш заказ']")))
await p.getByRole('button', { name: 'Оплатить заказ' }).click()
await p.waitForTimeout(700)
out('empty submit -> alerts', JSON.stringify((await p.getByRole('alert').allInnerTexts()).filter(Boolean)))
await p.getByLabel('Имя').fill('Анна')
await p.getByLabel('Улица и дом').fill('Гоголя 10')
await p.getByRole('button', { name: 'Оплатить заказ' }).click()
await p.waitForTimeout(700)
out(
	'typed name + street kept after error',
	`${await p.getByLabel('Имя').inputValue()} | ${await p.getByLabel('Улица и дом').inputValue()}`
)
out('remaining alerts', JSON.stringify((await p.getByRole('alert').allInnerTexts()).filter(Boolean)))

// address suggestions: calls DaData through /api/address-suggest, so it only runs with QA_DADATA=1 (and DADATA_API_KEY set)
if (process.env.QA_DADATA === '1') {
	// the open list is labelled «Улица и дом» too, so the input is found by its role
	const street = p.getByRole('combobox', { name: 'Улица и дом' })
	await street.fill('')
	await street.pressSequentially('Гоголя', { delay: 60 })
	await p
		.getByRole('option')
		.first()
		.waitFor({ timeout: 5000 })
		.catch(() => {})
	const options = await p.getByRole('option').allInnerTexts()
	out(
		'suggestions for «Гоголя»',
		`${options.length} options, all contain Гоголя: ${options.every((o) => o.includes('Гоголя'))}`
	)
	await street.press('ArrowDown')
	await street.press('Enter')
	out('ArrowDown + Enter fills the field', await street.inputValue())
} else {
	out('address suggestions', 'skipped (set QA_DADATA=1 with DADATA_API_KEY to call DaData)')
}

// variants: outskirts, pickup, scheduled
await p.getByRole('button', { name: /Населённый пункт/ }).click()
const townOptions = p.getByRole('option')
out('towns from the backend', JSON.stringify(await townOptions.allInnerTexts()))
{
	const label = (await townOptions.first().innerText()).replace(/\s+/g, ' ')
	const price = Number(label.match(/\+([\d\s]+)\s*₽/)?.[1].replace(/\s/g, ''))
	const freeFrom = Number(label.match(/бесплатно от ([\d\s]+)\s*₽/)?.[1].replace(/\s/g, '') ?? 0)
	const rows = await p.locator('aside ul:visible > li:not(:has-text("Доставка")) > span:last-child').allInnerTexts()
	const subtotal = rows.reduce((sum, row) => sum + Number(row.replace(/\D/g, '')), 0)
	const expected = freeFrom > 0 && subtotal >= freeFrom ? 0 : price
	out(
		'selected town: delivery line + total (expected delivery ' + expected + ', total ' + (subtotal + expected) + ')',
		(await txt(p.locator('aside li', { hasText: 'Доставка' }))) +
			' || ' +
			(await txt(p.locator('aside div:visible', { hasText: /^К оплате/ }).last()))
	)
}
if ((await townOptions.count()) > 1) await townOptions.nth(1).click()
else await p.keyboard.press('Escape')
out(
	'second town: delivery line + total',
	(await txt(p.locator('aside li', { hasText: 'Доставка' }))) + ' || ' + (await txt(p.getByText('К оплате')))
)
await p.getByRole('button', { name: /Когда/ }).click()
await p.getByRole('option', { name: /Ко времени/ }).click()
await p.getByLabel('Время', { exact: true }).fill('23:45')
await p.getByRole('button', { name: 'Оплатить заказ' }).click()
await p.waitForTimeout(700)
out('scheduled 23:45 -> error', JSON.stringify((await p.getByRole('alert').allInnerTexts()).filter(Boolean)))
await p.getByText('Самовывоз').click()
out(
	'pickup: address fields hidden',
	String((await p.getByLabel('Улица и дом').count()) === 0 && (await p.getByLabel('Населённый пункт').count()) === 0)
)
out('pickup: delivery line', await txt(p.locator('aside li', { hasText: 'Доставка' })))

// tampered cart -> server refuses
const t = await pageWithCart([{ dishId: 'ghost-dish', variantId: 'x', quantity: 1 }])
await t.goto(base + '/checkout')
await t.waitForTimeout(500)
out('unknown dish in cart -> priced out, empty state', await txt(t.getByText('В корзине пока пусто')))
// a size the dish does not have: the client skips it, server too
const t2 = await pageWithCart([{ dishId: CART[0].dishId, variantId: '1kg', quantity: 1 }])
await t2.goto(base + '/checkout')
await t2.waitForTimeout(500)
out('unknown size of a real dish -> empty state', await txt(t2.getByText('В корзине пока пусто')))
// server-side check without the browser's own filtering
const res = await fetch(base + '/checkout', { method: 'GET' })
out('GET /checkout', res.status)

// failed payment
const f = await pageWithCart()
await f.goto(base + '/checkout?payment=failed')
await f.waitForTimeout(500)
out('?payment=failed -> alert', JSON.stringify((await f.getByRole('alert').allInnerTexts()).filter(Boolean)))

// valid order: creates a real order on the backend, so it only runs with QA_PLACE_ORDER=1
if (process.env.QA_PLACE_ORDER === '1') {
	await p.getByText('Доставка', { exact: true }).first().click()
	await p.getByLabel('Улица и дом').fill('Гоголя 10')
	await p.getByRole('button', { name: /Когда/ }).click()
	await p.getByRole('option', { name: /В ближайшее/ }).click()
	await p.getByLabel('Телефон').fill('')
	await p.getByLabel('Телефон').pressSequentially('89183119791')
	await p.getByLabel(/Согласен/).check()
	await p.getByRole('button', { name: 'Оплатить заказ' }).click()
	await p.waitForTimeout(3000)
	const alerts = (await p.getByRole('alert').allInnerTexts()).filter(Boolean)
	out('after valid order: host', alerts.length ? 'no redirect' : new URL(p.url()).host)
	if (alerts.length) out('alerts', JSON.stringify(alerts))
} else {
	out('valid order', 'skipped (set QA_PLACE_ORDER=1 to create a real order)')
}

// empty cart
const e = await pageWithCart(null)
await e.goto(base + '/checkout')
await e.waitForTimeout(500)
out(
	'empty cart state + link',
	`${await txt(e.getByText('В корзине пока пусто'))} -> ${await e.getByRole('link', { name: 'Перейти в меню' }).getAttribute('href')}`
)
out('empty cart: no form / no submit', String(await e.locator('form#checkout').count()))

// ---------- contact form
const c = await (await browser.newContext({ viewport: { width: 1440, height: 1000 } })).newPage()
await c.goto(base + '/contacts')
await c.waitForTimeout(500)
await c.getByRole('button', { name: 'Отправить', exact: true }).click()
await c.waitForTimeout(700)
out('empty contact form -> alerts', JSON.stringify((await c.getByRole('alert').allInnerTexts()).filter(Boolean)))
await c.getByLabel('Имя').fill('Иван')
await c.getByLabel('Телефон или e-mail').fill('не телефон')
await c.getByLabel('Сообщение').fill('Здравствуйте, есть вопрос')
await c.getByRole('button', { name: 'Отправить', exact: true }).click()
await c.waitForTimeout(700)
out(
	'bad contact -> error, message kept',
	`${await txt(c.getByRole('alert').filter({ hasText: /\S/ }))} | ${await c.getByLabel('Сообщение').inputValue()}`
)
await c.getByText('Банкет', { exact: true }).click()
await c.getByLabel('Телефон или e-mail').fill('ivan@example.com')
await c.getByRole('button', { name: 'Отправить', exact: true }).click()
await c.waitForTimeout(800)
out('valid contact -> success', await txt(c.getByRole('status')))
// honeypot
const h = await (await browser.newContext()).newPage()
await h.goto(base + '/contacts')
await h.waitForTimeout(400)
await h.getByLabel('Имя').fill('Бот')
await h.getByLabel('Телефон или e-mail').fill('bot@spam.com')
await h.getByLabel('Сообщение').fill('buy now buy now')
await h.locator('input[name=website]').fill('http://spam', { force: true })
await h.getByRole('button', { name: 'Отправить', exact: true }).click()
await h.waitForTimeout(800)
out('honeypot filled -> looks like success', await txt(h.getByRole('status')))

// ---------- 404
const status = async (path) => (await fetch(base + path)).status
out('GET /nope', await status('/nope'))
const a = await browser.newPage()
await a.goto(base + '/nope')
out(
	'404 page h1 + home link',
	`${await txt(a.locator('h1'))} | ${await a.getByRole('link', { name: 'На главную' }).getAttribute('href')}`
)
await browser.close()
