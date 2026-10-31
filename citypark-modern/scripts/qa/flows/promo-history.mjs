// Production build + existing backend. QA_PROMO_WRITES=1 requires owner approval:
// creates unpaid orders and a temporary campaign, then edits/disables it in finally.
// No callbacks, provider navigation, emails, notifications or iiko requests.
// Optional QA_CUSTOMER_SESSION and QA_MINI_SESSION enable owned history checks.
// QA_CUSTOMER_SESSION_2 enables foreign-owner isolation. Without sessions, staff can
// inspect the website detail only; this is explicitly NOT a customer-auth acceptance.
// Read-only mode: QA_PROMO_ORDER_IDS=1,2 QA_LEGACY_ORDER_ID=3 (existing fixtures).
import assert from 'node:assert/strict'
import { chromium, expect } from 'playwright/test'

const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/+$/, '')
const api = process.env.MENU_API_URL?.replace(/\/+$/, '')
assert(api, 'Set MENU_API_URL to the existing configured backend')
const money = (value) => `${new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 }).format(value)}\u00a0₽`
const kopecks = (value) => Math.round(value * 100)
let admin
let campaign
const saved = []
const sessions = [
	{ name: 'web', path: '/account/orders', cookie: '__Host-citypark_customer', token: process.env.QA_CUSTOMER_SESSION },
	{ name: 'mini', path: '/mini/orders', cookie: '__Host-citypark_mini', token: process.env.QA_MINI_SESSION },
]
async function request(path, { token, body, method = body === undefined ? 'GET' : 'POST', status = 200 } = {}) {
	const response = await fetch(api + path, {
		method,
		headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
		...(body === undefined ? {} : { body: JSON.stringify(body) }),
	})
	assert.equal(response.status, status, `${method} ${path}: unexpected status ${response.status}`)
	return { response, data: await response.json() }
}
function arithmetic(order) {
	assert.equal(kopecks(order.subtotal) - kopecks(order.discount), kopecks(order.total))
	assert.equal(
		order.items.reduce((sum, item) => sum + kopecks(item.offeredPrice) * item.amount, 0),
		kopecks(order.subtotal)
	)
	assert.equal(
		order.items.reduce((sum, item) => sum + kopecks(item.discount), 0),
		kopecks(order.discount)
	)
}
const browser = await chromium.launch()
try {
	admin = (
		await request('/auth/sign-in', {
			body: {
				email: process.env.QA_ADMIN_EMAIL ?? 'dev@dev.io',
				password: process.env.QA_ADMIN_PASSWORD ?? 'devdevdev',
			},
		})
	).data.token
	if (process.env.QA_PROMO_WRITES === '1') {
		const categories = new Set((await request('/categories')).data.map((category) => category.id))
		const products = (await request('/products?pageSize=500')).data
			.filter((product) => categories.has(product.categoryId) && product.isAvailableForOrder && product.price > 10)
			.slice(0, 2)
		assert.equal(products.length, 2, 'Need two available dishes')
		const city = (await request('/delivery-prices')).data.find((town) => town.price > 0 && town.freeFrom === 0)?.name
		assert(city, 'Need a paid-delivery town')
		campaign = (
			await request('/promo-codes', {
				token: admin,
				status: 201,
				body: {
					code: `QHISTORY_${Date.now()}`,
					percentage: 50,
					enabled: true,
					target: 'selected',
					productIds: [products[0].id],
					categoryIds: [],
					minimumEligibleSubtotal: null,
					maximumDiscount: 0.01,
					startsAt: null,
					endsAt: null,
				},
			})
		).data
		const owners = [{ name: 'guest', token: undefined }, ...sessions.filter((surface) => surface.token)]
		for (const owner of owners)
			for (const delivery of [false, true]) {
				const items = products.map((product) => ({ productId: product.id, amount: 2 }))
				const quote = (
					await request('/orders/quote', {
						body: {
							items,
							fulfillment: delivery ? 'delivery' : 'pickup',
							...(delivery ? { city } : {}),
							promoCode: campaign.code,
						},
					})
				).data
				assert.equal(quote.discount, 0.01, 'Capped kopeck fixture')
				const { data: created, response } = await request('/orders', {
					token: owner.token,
					status: 201,
					body: {
						items,
						name: `Тест история ${owner.name} ${Date.now()}`,
						phone: `+7999${String(Date.now()).slice(-7)}`,
						comment: 'QA: неоплаченный тест, не готовить',
						...(delivery ? { address: { city, address: 'Тестовый адрес, не доставлять' } } : {}),
						promoCode: campaign.code,
						expectedPayableTotal: quote.payableTotal,
					},
				})
				assert.equal(
					kopecks(Number(new URL(created.redirectUrl).searchParams.get('MNT_AMOUNT'))),
					kopecks(quote.payableTotal),
					'MONETA signed URL includes saved delivery'
				)
				const snapshot = (await request(`/orders/${created.orderId}`, { token: admin })).data
				assert.equal(snapshot.status, 'pending', 'Orders must remain unpaid')
				assert.equal(kopecks(snapshot.total) + kopecks(snapshot.deliveryPrice), kopecks(quote.payableTotal))
				arithmetic(snapshot)
				saved.push({ order: snapshot, owner: owner.name })
				console.log(`Created unpaid ${owner.name} order ${snapshot.id}`)
				// Guest retry scope must never grant access to authenticated order history.
				if (!owner.token) {
					const scope = /^citypark_guest_checkout=([^;]+)/.exec(response.headers.get('set-cookie') ?? '')?.[1]
					assert(scope, 'Guest response must include ownership scope')
					const statusResponse = await fetch(`${api}/orders/${snapshot.id}`, {
						headers: { Cookie: `citypark_guest_checkout=${scope}` },
					})
					assert.equal(statusResponse.status, 401)
				}
			}
		campaign = (
			await request(`/promo-codes/${campaign.id}`, {
				token: admin,
				method: 'PUT',
				body: { ...campaign, enabled: false, percentage: 99, maximumDiscount: 100 },
			})
		).data
		console.log(
			'Guest/available-session pickup+delivery: capped kopeck allocation, MONETA payable, pending status; campaign edited+disabled PASS'
		)
	} else {
		assert(process.env.QA_PROMO_ORDER_IDS, 'Supply QA_PROMO_ORDER_IDS or approve QA_PROMO_WRITES=1')
		for (const id of process.env.QA_PROMO_ORDER_IDS.split(','))
			saved.push({ order: (await request(`/orders/${id}`, { token: admin })).data, owner: 'fixture' })
	}
	const list = (await request('/orders?pageSize=500', { token: admin })).data
	const legacy = process.env.QA_LEGACY_ORDER_ID
		? (await request(`/orders/${process.env.QA_LEGACY_ORDER_ID}`, { token: admin })).data
		: list.orders.find((order) => order.promoCode === null)
	assert(legacy, 'Need a saved no-code order')
	assert.equal(legacy.promoCode, null)
	saved.push({ order: legacy, owner: 'legacy' })
	for (const { order } of saved) {
		assert.deepEqual(
			(await request(`/orders/${order.id}`, { token: admin })).data,
			order,
			'Snapshot survives campaign edit/disable'
		)
		arithmetic(order)
	}
	const surfaces = [
		{ name: 'staff inspection of website', path: '/account/orders', cookie: '__Host-citypark_customer', token: admin },
		...sessions.filter((surface) => surface.token),
	]
	for (const surface of surfaces)
		for (const width of [390, 1440])
			for (const theme of ['light', 'dark']) {
				const context = await browser.newContext({ viewport: { width, height: 900 } })
				await context.addInitScript((theme) => localStorage.setItem('theme', theme), theme)
				await context.addCookies([
					{
						name: surface.cookie,
						value: surface.token,
						url: base.replace(/^http:/, 'https:'),
						secure: true,
						httpOnly: true,
						sameSite: 'Lax',
					},
				])
				const page = await context.newPage()
				if (surface.name === 'mini')
					await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
						route.fulfill({
							contentType: 'application/javascript',
							body: 'window.Telegram={WebApp:{initData:"read-only",ready(){},expand(){}}}',
						})
					)
				if (surface.name === 'web' || surface.name === 'mini') {
					const owned = (await request('/orders/my?pageSize=500', { token: surface.token })).data
					for (const fixture of saved.filter((fixture) => fixture.owner === surface.name))
						assert(
							owned.orders.some((order) => order.id === fixture.order.id),
							'Own order must appear in history'
						)
					for (const fixture of saved.filter((fixture) => fixture.owner === 'guest'))
						assert(
							!owned.orders.some((order) => order.id === fixture.order.id),
							'Guest orders cannot appear in account history'
						)
					await page.goto(base + surface.path)
					for (const { order } of saved.filter((fixture) => fixture.owner === surface.name)) {
						const row = page
							.getByRole('listitem')
							.filter({ has: page.getByRole('link', { name: `Заказ № ${order.id}`, exact: true }) })
						await expect(row).toContainText(money((kopecks(order.total) + kopecks(order.deliveryPrice)) / 100))
					}
				}
				let browserBackendCalls = 0
				page.on('request', (request) => {
					if (request.url().startsWith(api)) browserBackendCalls++
				})
				for (const { order, owner } of saved) {
					if (surface.name === 'web' || surface.name === 'mini') {
						const actual = await fetch(`${api}/orders/${order.id}`, {
							headers: { Authorization: `Bearer ${surface.token}` },
						})
						if (actual.status === 404) {
							assert.notEqual(owner, surface.name, 'Own order must remain readable')
							await page.goto(`${base}${surface.path}/${order.id}`)
							await expect(page.getByRole('heading', { name: `Заказ № ${order.id}`, exact: true })).toHaveCount(0)
							continue
						}
						assert.equal(actual.status, 200)
					}
					const response = await page.goto(`${base}${surface.path}/${order.id}`)
					assert.match(response.headers()['cache-control'], /private.*no-store/)
					await expect(page.getByRole('heading', { name: `Заказ № ${order.id}` })).toBeVisible()
					if (order.promoCode) {
						const totals = page.locator('dl[aria-label="Итоги заказа"] > div')
						const expected = [
							['Блюда до скидки:', order.subtotal],
							['Скидка по промокоду:', 0 - order.discount],
							['Блюда со скидкой:', order.total],
							['Доставка:', order.deliveryPrice],
							['Итого:', (kopecks(order.total) + kopecks(order.deliveryPrice)) / 100],
						]
						for (const [label, amount] of expected)
							await expect(
								totals.filter({ has: page.locator('dt').filter({ hasText: new RegExp(`^${label}$`) }) }).locator('dd')
							).toHaveText(money(amount))
						await expect(page.getByText(order.promoCode, { exact: true })).toBeVisible()
						const rows = page.locator('main ul > li')
						for (const [index, item] of order.items.entries()) {
							const original = kopecks(item.offeredPrice) * item.amount
							await expect(rows.nth(index).locator('p').nth(0)).toHaveText(money(original / 100))
							await expect(rows.nth(index).locator('p').nth(1)).toHaveText(`Скидка: ${money(0 - item.discount)}`)
							await expect(rows.nth(index).locator('p').nth(2)).toHaveText(
								`Со скидкой: ${money((original - kopecks(item.discount)) / 100)}`
							)
						}
					} else {
						await expect(page.getByText(/^Товары:/)).toBeVisible()
						await expect(page.getByText('Скидка по промокоду:', { exact: true })).toHaveCount(0)
						await expect(page.getByText(/^Итого:/)).toHaveText(
							`Итого: ${money(Math.round(order.total + order.deliveryPrice))}`
						)
					}
					assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
					await page.screenshot({
						path: `/tmp/citypark-promo-history-${surface.name.split(' ')[0]}-${order.id}-${width}-${theme}.png`,
						fullPage: true,
					})
				}
				assert.equal(browserBackendCalls, 0, 'Backend calls stay server-only')
				await page.screenshot({
					path: `/tmp/citypark-promo-history-${surface.name.split(' ')[0]}-${width}-${theme}.png`,
					fullPage: true,
				})
				console.log(
					`${surface.name} ${width}px ${theme}: snapshots, legacy and line/header arithmetic expected=actual PASS`
				)
				await context.close()
			}
	for (const token of [process.env.QA_CUSTOMER_SESSION_2, ...sessions.map((surface) => surface.token)].filter(
		Boolean
	)) {
		for (const { order, owner } of saved.filter(
			(fixture) =>
				fixture.owner === 'guest' ||
				(token === process.env.QA_CUSTOMER_SESSION_2 && fixture.owner !== 'legacy' && fixture.owner !== 'fixture')
		)) {
			await request(`/orders/${order.id}`, { token, status: 404 })
			console.log(`Ownership excludes ${owner} order ${order.id}: PASS`)
		}
	}
	for (const surface of sessions)
		if (!surface.token)
			console.log(
				`PENDING: ${surface.name} authenticated checkout/history/ownership needs ${surface.name === 'web' ? 'QA_CUSTOMER_SESSION' : 'QA_MINI_SESSION'}`
			)
	if (!process.env.QA_CUSTOMER_SESSION_2) console.log('PENDING: second-account isolation needs QA_CUSTOMER_SESSION_2')
	console.log(
		`Unpaid QA order IDs: ${saved
			.filter((fixture) => fixture.owner !== 'legacy')
			.map((fixture) => fixture.order.id)
			.join(', ')}`
	)
	console.log(
		'PENDING release acceptance: authenticated Telegram device handoff/payment return; real MONETA charge/callback; restaurant iiko discounted food reconciliation and staff notifications. Never activate campaigns before these checks.'
	)
} finally {
	if (campaign)
		await request(`/promo-codes/${campaign.id}`, { token: admin, method: 'PUT', body: { ...campaign, enabled: false } })
	await browser.close()
}
