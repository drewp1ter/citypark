// Geolocation QA uses intercepted coordinates and provider responses; it never creates orders or contacts DaData.
// Optional authenticated Telegram check requires QA_MINI_SESSION and a device capable of granting geolocation permission.
import assert from 'node:assert/strict'
import { chromium, expect } from 'playwright/test'
import { postAddressGeolocate } from '../../../src/app/api/address-geolocate-handler.ts'
import { matchDeliveryTown } from '../../../src/entities/dish/lib/match-delivery-town.ts'
import { seedCart } from '../menu-cart.mjs'

const towns = [
	{ name: 'Белореченск', price: 100, freeFrom: 500 },
	{ name: 'п. Мирный', price: 150, freeFrom: 0 },
	{ name: 'п. Семёновка', price: 100, freeFrom: 0 },
]
const address = {
	region: 'Краснодарский',
	region_type: 'край',
	city: 'Белореченск',
	settlement: null,
	street_with_type: 'ул Гоголя',
	house_type: 'д',
	house: '61',
	block_type: 'к',
	block: '2',
}
const deps = (overrides = {}) => ({
	key: () => 'test-key',
	geocode: async () => address,
	towns: async () => towns,
	match: matchDeliveryTown,
	...overrides,
})
const post = (body, dependencies = deps()) =>
	postAddressGeolocate(
		new Request('http://local/api/address-geolocate', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body,
		}),
		dependencies
	)

for (const [value, expected] of [
	['Г. Белореченск', 'Белореченск'],
	['бЕЛОРЕЧЕНСК', 'Белореченск'],
	['п. Мирный', 'п. Мирный'],
	['семеновка', 'п. Семёновка'],
]) {
	assert.equal(matchDeliveryTown(value, towns)?.name, expected)
}
assert.equal(matchDeliveryTown('г. Белореченск', [...towns, { name: 'Белореченск' }]), undefined)
assert.equal(matchDeliveryTown('Белореченск', [{ name: 'Краснодар' }]), undefined)
assert.equal(matchDeliveryTown('Белореченск', [{ name: 'г. Белореченск' }, { name: 'Белореченск' }]), undefined)
const matched = await post('{"lat":44.77,"lon":39.87}')
assert.equal(matched.status, 200)
assert.equal(matched.headers.get('Cache-Control'), 'no-store')
assert.deepEqual(await matched.json(), { status: 'matched', town: 'Белореченск', street: 'ул Гоголя, д 61, к 2' })
for (const body of [
	'{',
	'{"lat":"44.77","lon":39.87}',
	'{"lat":44.77}',
	'{"lat":91,"lon":39.87}',
	'{"lat":44.77,"lon":181}',
]) {
	assert.equal((await post(body)).status, 400, `invalid request ${body}`)
}
assert.equal((await post('{"lat":44,"lon":39}', deps({ key: () => undefined }))).status, 503)
assert.equal(
	(
		await post(
			'{"lat":44,"lon":39}',
			deps({
				geocode: async () => {
					throw new Error('provider')
				},
			})
		)
	).status,
	502
)
assert.deepEqual(
	await (
		await post(
			'{"lat":44,"lon":39}',
			deps({ geocode: async () => ({ ...address, settlement: 'Несуществующий', city: 'Белореченск' }) })
		)
	).json(),
	{ status: 'unmatched' }
)
assert.deepEqual(
	await (
		await post('{"lat":44,"lon":39}', deps({ geocode: async () => ({ ...address, region: 'Ростовский' }) }))
	).json(),
	{ status: 'unmatched' }
)
assert.deepEqual(
	await (
		await post(
			'{"lat":44,"lon":39}',
			deps({ geocode: async () => ({ ...address, street_with_type: null, house: null, block: null }) })
		)
	).json(),
	{ status: 'matched', town: 'Белореченск', street: '' }
)
console.log('Geolocation route handler: validation, matching, no-store and provider failures PASS')

const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/$/, '')
const cart = await seedCart([1])
const browser = await chromium.launch()
try {
	for (const width of [390, 1440])
		for (const theme of ['light', 'dark']) {
			const context = await browser.newContext({ viewport: { width, height: 950 }, colorScheme: theme })
			await context.grantPermissions(['geolocation'])
			await context.setGeolocation({ latitude: 44.77, longitude: 39.87 })
			await context.addInitScript(
				({ cart, theme }) => {
					localStorage.setItem('citypark-cart', JSON.stringify(cart))
					localStorage.setItem('theme', theme)
				},
				{ cart, theme }
			)
			const page = await context.newPage()
			let requests = 0
			await page.route('**/api/address-geolocate', async (route) => {
				requests++
				await route.fulfill({ json: { status: 'matched', town: 'Белореченск', street: 'ул Гоголя, д 61' } })
			})
			await page.goto(base + '/checkout')
			await expect(page.getByRole('status').filter({ hasText: 'Адрес определён.' })).toBeVisible({ timeout: 12000 })
			await expect(page.getByLabel('Улица и дом')).toHaveValue('ул Гоголя, д 61')
			await expect(page.getByRole('button', { name: /^Оплатить/ })).toBeEnabled({ timeout: 10000 })
			await expect.poll(() => requests).toBe(1)
			await page.getByLabel('Улица и дом').fill('Мой адрес')
			await expect(page.getByLabel('Улица и дом')).toHaveValue('Мой адрес')
			await context.close()
			console.log(`Website autofill ${width}px ${theme}: PASS`)
		}

	// A delayed response must not override edits made while the request is in flight.
	const context = await browser.newContext({ viewport: { width: 390, height: 900 } })
	await context.grantPermissions(['geolocation'])
	await context.setGeolocation({ latitude: 44.77, longitude: 39.87 })
	await context.addInitScript((cart) => localStorage.setItem('citypark-cart', JSON.stringify(cart)), cart)
	const page = await context.newPage()
	let release
	await page.route('**/api/address-geolocate', async (route) => {
		await new Promise((resolve) => {
			release = resolve
		})
		await route.fulfill({ json: { status: 'matched', town: 'Белореченск', street: 'ул Гоголя, д 61' } })
	})
	await page.goto(base + '/checkout')
	await expect(page.getByRole('status').filter({ hasText: 'Определяем ваш адрес' })).toBeVisible({ timeout: 10000 })
	await page.getByLabel('Улица и дом').fill('Введённая улица')
	release()
	await page.waitForTimeout(100)
	await expect(page.getByLabel('Улица и дом')).toHaveValue('Введённая улица')
	await context.close()
	console.log('Delayed autofill preserves manual address edits: PASS')

	const invalid = await fetch(`${base}/api/address-geolocate`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: '{',
	})
	assert.equal(invalid.status, 400)
	assert.deepEqual(await invalid.json(), { status: 'invalid' })
	assert.equal(invalid.headers.get('cache-control'), 'no-store')
	console.log('Production route malformed JSON response: PASS')
	if (!process.env.QA_MINI_SESSION)
		console.log(
			'Authenticated Telegram Mini App geolocation: PENDING (supply QA_MINI_SESSION and device permission access)'
		)
} finally {
	await browser.close()
}
