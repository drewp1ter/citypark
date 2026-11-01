// QA_DELIVERY_WRITES=1 authorizes temporary settings/town writes, restored in finally. No orders.
import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import path from 'node:path'
import ts from 'typescript'
import { chromium, expect } from 'playwright/test'
import { seedCart } from '../menu-cart.mjs'
const base = process.env.BASE_URL ?? 'http://localhost:3111'
const api = process.env.MENU_API_URL?.replace(/\/+$/, '')
assert(api, 'MENU_API_URL required')
// Execute the actual pure TypeScript helpers, with only their local formatting dependency.
async function load(file) {
	let code = ts.transpileModule(await fs.readFile(file, 'utf8'), {
		compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
	}).outputText
	for (const match of [...code.matchAll(/from ['"]([^'"]+)['"]/g)]) {
		const target =
			match[1] === '@/shared/lib'
				? path.resolve('src/shared/lib/format.ts')
				: path.resolve(path.dirname(file), match[1] + '.ts')
		code = code.replace(match[0], `from '${await load(target)}'`)
	}
	return 'data:text/javascript;base64,' + Buffer.from(code).toString('base64')
}
const { formatDeliveryInfo, selectDeliveryCity } = await import(
	await load(path.resolve('src/entities/dish/lib/format-delivery-info.ts'))
)
const { quoteDelivery } = await import(await load(path.resolve('src/entities/dish/lib/quote-delivery.ts')))
const { formatPrice, typo } = await import(await load(path.resolve('src/shared/lib/format.ts')))
const town = { name: 'г. Белореченск', price: 150, freeFrom: 600 }
const input = { cooking: '~35 мин', outskirtsLabel: 'СОТ № 3', outskirtsMin: 200, outskirtsMax: 350 }
for (const [min, max, label] of [
	[200, 350, '200–350 ₽'],
	[0, 0, '0 ₽'],
	[100000, 100000, '100 000 ₽'],
]) {
	const info = formatDeliveryInfo({ ...input, outskirtsMin: min, outskirtsMax: max }, [town])
	assert.equal(info.outskirtsRange, label)
	assert.equal(info.cooking, typo(input.cooking))
	assert.equal(info.outskirtsFee, typo(`${input.outskirtsLabel} ${label}`))
}
const info = formatDeliveryInfo(input, [town])
for (const [subtotal, amount] of [
	[599, 150],
	[600, 0],
	[601, 0],
])
	assert.deepEqual(quoteDelivery({ method: 'delivery', zone: 'city', subtotal }, info), {
		amount,
		label: formatPrice(amount),
	})
assert.equal(quoteDelivery({ method: 'pickup', zone: 'city', subtotal: 0 }, info).amount, 0)
assert.equal(quoteDelivery({ method: 'delivery', zone: 'outskirts', subtotal: 0 }, info).label, info.outskirtsRange)
assert.equal(
	quoteDelivery({ method: 'delivery', zone: 'city', subtotal: 1000 }, { ...info, city: { ...town, freeFrom: 0 } })
		.amount,
	150
)
for (const towns of [
	[],
	[town, town],
	[{ ...town, price: -1 }],
	[{ ...town, freeFrom: NaN }],
	[{ ...town, freeFrom: '600' }],
])
	assert.equal(selectDeliveryCity(towns), null)
assert.equal(
	quoteDelivery({ method: 'delivery', zone: 'city', subtotal: 1000 }, formatDeliveryInfo(input, [])).label,
	'уточним'
)
for (const invalid of [
	null,
	{},
	{ ...input, cooking: ' ' },
	{ ...input, cooking: 'я'.repeat(101) },
	{ ...input, outskirtsLabel: 'я'.repeat(61) },
	{ ...input, outskirtsMin: 1.5 },
	{ ...input, outskirtsMax: 100001 },
	{ ...input, outskirtsMin: 400 },
])
	assert.equal(formatDeliveryInfo(invalid, [town]).cooking, typo('~30 мин'))
console.log('PASS pure formatting, selection, fallbacks and quotes')
let token
async function backend(endpoint, body, method = 'PUT') {
	const r = await fetch(api + endpoint, {
		method: body === undefined ? 'GET' : method,
		headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
		...(body === undefined ? {} : { body: JSON.stringify(body) }),
	})
	const value = await r.json()
	assert(r.ok, `${endpoint}: ${r.status} ${JSON.stringify(value)}`)
	return value
}
const original = await backend('/delivery-settings')
const towns = await backend('/delivery-prices')
const city = towns.find((t) => t.name.trim() === 'Белореченск')
assert(city, 'Need Белореченск tariff')
const cart = await seedCart([1])
const browser = await chromium.launch()
let changed = false
async function verify(settings, tariff, all = true) {
	const expected = formatDeliveryInfo(settings, [tariff])
	for (const width of all ? [390, 1440] : [1440])
		for (const theme of all ? ['light', 'dark'] : ['light']) {
			const context = await browser.newContext({ viewport: { width, height: 1000 } })
			await context.addInitScript(
				({ cart, theme }) => {
					localStorage.setItem('citypark-cart', JSON.stringify(cart))
					localStorage.setItem('theme', theme)
				},
				{ cart, theme }
			)
			const page = await context.newPage()
			try {
				await page.goto(base + '/menu')
				await expect(page.getByText(expected.cooking, { exact: false }).first()).toBeVisible()
				const menuText = (await page.locator('main').innerText()).replaceAll('\u00a0', ' ')
				if (tariff.freeFrom > 0)
					assert(menuText.includes(`0 ₽ от ${formatPrice(tariff.freeFrom).replaceAll('\u00a0', ' ')}`))
				else assert(!/0 ₽ от/.test(menuText))
				if (width === 1440) await expect(page.getByText(expected.outskirtsFee, { exact: true })).toBeVisible()
				let summary = page.locator('aside[aria-label="Корзина"]')
				if (width === 390) {
					await page.locator("nav[aria-label='Корзина'] button").click()
					summary = page.locator('dialog.cart-sheet')
				}
				await expect(summary).toBeVisible()
				// Real cart food subtotal determines the live tariff estimate.
				const totalText = await summary.innerText()
				assert(totalText.includes('Доставка'))
				const products = await backend('/products?pageSize=500')
				const product = products.find((p) => String(p.id) === cart[0].dishId)
				const amount = tariff.freeFrom > 0 && product.price >= tariff.freeFrom ? 0 : tariff.price
				await expect(summary.getByText(formatPrice(amount), { exact: true }).first()).toBeVisible()
				await page.goto(base + '/')
				const home = (await page.locator('main').innerText()).replaceAll('\u00a0', ' ')
				if (width === 1440) assert(home.includes(expected.outskirtsFee.replaceAll('\u00a0', ' ')))
				if (tariff.freeFrom === 0) assert(home.includes('Доставка еды'))
				else assert(home.includes(formatPrice(tariff.freeFrom).replaceAll('\u00a0', ' ')))
			} finally {
				await context.close()
			}
		}
}
async function poll(settings, tariff) {
	const end = Date.now() + 240000
	while (true) {
		try {
			await verify(settings, tariff, false)
			return
		} catch (e) {
			if (Date.now() > end) throw e
			await new Promise((resolve) => setTimeout(resolve, 5000))
		}
	}
}
try {
	await verify(original, city)
	if (process.env.QA_DELIVERY_WRITES === '1') {
		token = (
			await backend(
				'/auth/sign-in',
				{ email: process.env.QA_ADMIN_EMAIL ?? 'dev@dev.io', password: process.env.QA_ADMIN_PASSWORD ?? 'devdevdev' },
				'POST'
			)
		).token
		changed = true
		const updated = {
			...original,
			cooking: '~37 мин',
			outskirtsLabel: 'СОТ и ДНТ',
			outskirtsMin: 210,
			outskirtsMax: 360,
		}
		await backend('/delivery-settings', updated)
		await poll(updated, city)
		await verify(updated, city)
		const zero = { ...city, freeFrom: 0 }
		await backend('/delivery-prices/' + city.id, zero, 'PATCH')
		await poll(updated, zero)
		await verify(updated, zero)
		const free = { ...city, freeFrom: 1 }
		await backend('/delivery-prices/' + city.id, free, 'PATCH')
		await poll(updated, free)
		await verify(updated, free)
		console.log('PASS independent settings/tariff refresh, zero threshold and free cart estimate')
	} else console.log('PENDING live-edit cache checks: set QA_DELIVERY_WRITES=1 with owner authorization')
	assert((await fs.readFile('app/layout.tsx', 'utf8')).includes('DeliveryProvider'))
	assert((await fs.readFile('src/widgets/cart-summary/ui/cart-sheet.tsx', 'utf8')).includes('useDeliveryInfo'))
	console.log('PASS responsive live facts and shared Mini App provider path; authenticated device checks remain manual')
} finally {
	if (changed) {
		await backend('/delivery-settings', original)
		await backend('/delivery-prices/' + city.id, city, 'PATCH')
		console.log('Restored original backend values')
		await poll(original, city)
	}
	await browser.close()
}
