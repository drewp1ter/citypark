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
	region_iso_code: 'RU-KDA',
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

for (const name of ['Майкоп', 'п. Гончарка']) {
	const adTown = { name, regionCode: 'RU-AD', price: 100, freeFrom: 0 }
	const geocode = async () => ({
		...address,
		region_iso_code: 'RU-AD',
		region: 'Адыгея',
		region_type: 'Респ',
		city: name,
		settlement: null,
	})
	const result = await post('{"lat":44,"lon":39}', deps({ geocode, towns: async () => [adTown] }))
	assert.equal((await result.json()).status, 'matched')
	const mismatch = await post(
		'{"lat":44,"lon":39}',
		deps({ geocode, towns: async () => [{ ...adTown, regionCode: 'RU-KDA' }] })
	)
	assert.equal((await mismatch.json()).status, 'unmatched')
}
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
		await post(
			'{"lat":44,"lon":39}',
			deps({ geocode: async () => ({ ...address, region_iso_code: 'RU-ROS', region: 'Ростовский' }) })
		)
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
// Every handled failure has exactly one structured report and leaves its API contract intact.
for (const [overrides, reason] of [
	[{ key: () => undefined }, 'configuration'],
	[{ geocode: async () => null }, 'empty_results'],
	[{ geocode: async () => ({ ...address, region_iso_code: 'other', region: 'other' }) }, 'unsupported_region'],
	[{ towns: async () => [] }, 'no_town'],
	[{ towns: async () => [towns[0], towns[0]] }, 'ambiguous_town'],
	[
		{
			towns: async () => {
				throw new Error('private backend payload')
			},
		},
		'delivery_list',
	],
	[
		{
			match: () => {
				throw new Error('private matching payload')
			},
		},
		'town_matching',
	],
	[{ geocode: async () => ({ ...address, street_with_type: null }) }, 'partial'],
	...['provider_network', 'provider_timeout', 'provider_http', 'provider_json', 'provider_shape'].map((reason) => [
		{
			geocode: async () => {
				throw Object.assign(new Error('private provider payload'), { reason, httpStatus: 403 })
			},
		},
		reason,
	]),
]) {
	const reports = []
	await post(
		'{"lat":44,"lon":39}',
		deps({ ...overrides, normalize: (value) => value.toLowerCase(), report: (...args) => reports.push(args) })
	)
	assert.equal(reports.length, 1)
	assert.equal(reports[0][0], reason)
	assert.equal(JSON.stringify(reports).includes('private'), false)
	assert.ok(reports[0][1].elapsed_ms >= 0)
}
const reports = []
await post('{', deps({ report: (...args) => reports.push(args) }))
assert.equal(reports[0][0], 'invalid')
reports.length = 0
await post('{"lat":44,"lon":39}', deps({ report: (...args) => reports.push(args) }))
assert.equal(reports.length, 0)
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
			await context.route('**/api/address-suggest?**', (route) => route.fulfill({ json: [] }))
			const page = await context.newPage()
			let requests = 0
			await page.route('**/api/address-geolocate', async (route) => {
				requests++
				await route.fulfill({ json: { status: 'matched', town: 'Белореченск', street: 'ул Гоголя, д 61' } })
			})
			await page.goto(base + '/checkout')
			await expect(page.getByRole('status').filter({ hasText: 'Адрес определён.' })).toBeVisible({ timeout: 12000 })
			await expect(page.getByRole('combobox', { name: 'Улица и дом' })).toHaveValue('ул Гоголя, д 61')
			await expect(page.getByRole('button', { name: /^Оплатить/ })).toBeEnabled({ timeout: 10000 })
			await expect.poll(() => requests).toBe(1)
			await page.getByRole('combobox', { name: 'Улица и дом' }).fill('Мой адрес')
			await expect(page.getByRole('combobox', { name: 'Улица и дом' })).toHaveValue('Мой адрес')
			await context.close()
			console.log(`Website autofill ${width}px ${theme}: PASS`)
		}

	// A delayed response must not override edits made while the request is in flight.
	const context = await browser.newContext({ viewport: { width: 390, height: 900 } })
	await context.grantPermissions(['geolocation'])
	await context.setGeolocation({ latitude: 44.77, longitude: 39.87 })
	await context.addInitScript((cart) => localStorage.setItem('citypark-cart', JSON.stringify(cart)), cart)
	await context.route('**/api/address-suggest?**', (route) => route.fulfill({ json: [] }))
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
	await page.getByRole('combobox', { name: 'Улица и дом' }).fill('Введённая улица')
	release()
	await page.waitForTimeout(100)
	await expect(page.getByRole('combobox', { name: 'Улица и дом' })).toHaveValue('Введённая улица')
	await context.close()
	console.log('Delayed autofill preserves manual address edits: PASS')

	// A hydrated saved default wins before browser location can start.
	const saved = await browser.newContext({ viewport: { width: 390, height: 900 } })
	await saved.addInitScript((cart) => {
		localStorage.setItem('citypark-cart', JSON.stringify(cart))
		const entry = {
			id: crypto.randomUUID(),
			revision: crypto.randomUUID(),
			isDefault: true,
			name: 'Дом',
			city: 'Белореченск',
			street: 'Гоголя, 61',
			apartment: '12',
			entrance: '2',
		}
		localStorage.setItem(
			'citypark.addresses.v1',
			JSON.stringify({ version: 1, revision: crypto.randomUUID(), entries: [entry] })
		)
		window.locationRequests = 0
		Object.defineProperty(navigator, 'geolocation', {
			value: {
				getCurrentPosition() {
					window.locationRequests++
				},
			},
		})
	}, cart)
	const savedPage = await saved.newPage()
	await savedPage.route('**/api/address-suggest?**', (route) => route.fulfill({ json: [] }))
	await savedPage.goto(base + '/checkout')
	await expect(savedPage.getByRole('combobox', { name: 'Улица и дом' })).toHaveValue('Гоголя, 61')
	await expect(savedPage.locator('input[name="apartment"]')).toHaveValue('12')
	await expect(savedPage.locator('input[name="entrance"]')).toHaveValue('2')
	assert.equal(await savedPage.evaluate(() => window.locationRequests), 0)
	await saved.close()
	console.log('Guest hydration/default prevents geolocation and preserves all address fields: PASS')

	// A denied location request selects the first delivery option without filling the street.
	const denied = await browser.newContext()
	await denied.addInitScript((cart) => {
		localStorage.setItem('citypark-cart', JSON.stringify(cart))
		Object.defineProperty(navigator, 'geolocation', {
			value: { getCurrentPosition: (_success, failure) => failure({ code: 1 }) },
		})
	}, cart)
	const deniedPage = await denied.newPage()
	await deniedPage.goto(base + '/checkout')
	await expect(deniedPage.getByRole('status').filter({ hasText: 'Проверьте населённый пункт' })).toBeVisible()
	const city = deniedPage.locator('select[name="city"]')
	const firstTown = await city.locator('option:not([value=""])').first().getAttribute('value')
	await expect(city).toHaveValue(firstTown)
	await expect(deniedPage.getByRole('combobox', { name: 'Улица и дом' })).toHaveValue('')
	await denied.close()

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
