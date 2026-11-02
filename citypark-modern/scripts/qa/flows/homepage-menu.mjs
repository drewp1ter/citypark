// QA_HOMEPAGE_WRITES=1 temporarily curates the homepage, restoring it in finally. No orders or payments.
import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import ts from 'typescript'
import { chromium, expect } from 'playwright/test'

async function moduleUrl(file) {
	return (
		'data:text/javascript;base64,' +
		Buffer.from(
			ts.transpileModule(await fs.readFile(file, 'utf8'), {
				compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
			}).outputText
		).toString('base64')
	)
}
const { selectHomepageMenu } = await import(await moduleUrl('src/entities/dish/lib/select-homepage-menu.ts'))
const placeholder = '/images/placeholder.svg'
const dish = (id, categoryId, available = true, photo = placeholder, orderBlocked = false) => ({
	id: String(id),
	categoryId: String(categoryId),
	available,
	orderBlocked,
	name: `Блюдо ${id}`,
	photo: { src: photo, alt: '' },
	variants: [{ id: 'default', weight: '', price: 100 }],
})
const categories = Array.from({ length: 6 }, (_, i) => ({ id: String(i + 1), name: `Категория ${i + 1}` }))
const menu = {
	categories,
	dishes: [
		dish(1, 1, false, '/a.jpg'),
		dish(2, 1),
		dish(3, 1, true, '/a.jpg'),
		dish(4, 1, true, '/b.jpg'),
		dish(5, 2, false, placeholder, true),
		dish(6, 3),
		dish(7, 4),
		dish(8, 5),
	],
}
const resolve = (configuration, input = menu) => selectHomepageMenu(input, configuration, placeholder)
assert.deepEqual(
	resolve({ categories: [] }).map((g) => g.category.id),
	['1', '2', '3', '4']
)
assert.deepEqual(
	resolve({ categories: [] })[0].dishes.map((d) => d.id),
	['3', '4', '2']
)
assert.deepEqual(
	resolve({
		categories: [
			{ categoryId: 2, productIds: [5] },
			{ categoryId: 1, productIds: [2, 1] },
		],
	}).map((g) => g.dishes.map((d) => d.id)),
	[['5'], ['2', '1']]
)
assert.equal(resolve({ categories: [{ categoryId: 2, productIds: [5] }] })[0].dishes[0].orderBlocked, true)
assert.deepEqual(
	resolve({ categories: [{ categoryId: 1, productIds: [999, 2, 5] }] })[0].dishes.map((d) => d.id),
	['2']
)
assert.deepEqual(
	resolve({ categories: [{ categoryId: 1, productIds: [999, 5] }] })[0].dishes.map((d) => d.id),
	['3', '4', '2']
)
assert.deepEqual(
	resolve({
		categories: [
			{ categoryId: 999, productIds: [] },
			{ categoryId: 6, productIds: [] },
		],
	}),
	[]
)
assert.deepEqual(resolve({ categories: [] }, { categories: [], dishes: menu.dishes }), [])
console.log(
	'PASS resolver: defaults, stable ranking, manual ordering, no top-up, blocked/unavailable, stale/moved/hidden IDs and empty groups'
)
if (process.argv.includes('--pure')) process.exit(0)

process.loadEnvFile('.env')
const api = process.env.MENU_API_URL?.replace(/\/+$/, '')
assert(api, 'MENU_API_URL required')
const base = process.env.BASE_URL ?? 'http://localhost:3111'
let token
async function backend(path, body, method = 'GET') {
	const response = await fetch(api + path, {
		method,
		headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
		...(body ? { body: JSON.stringify(body) } : {}),
	})
	assert(response.ok, `${method} ${path}: ${response.status}`)
	return response.json()
}
const shared =
	'data:text/javascript;base64,' +
	Buffer.from(
		`export * from '${await moduleUrl('src/shared/lib/format.ts')}'; export * from '${await moduleUrl('src/shared/lib/photo.ts')}';`
	).toString('base64')
const mappingCode = ts
	.transpileModule(await fs.readFile('src/entities/dish/api/to-dish.ts', 'utf8'), {
		compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
	})
	.outputText.replace("from '@/shared/lib'", `from '${shared}'`)
const { toDish, toCategory } = await import(
	'data:text/javascript;base64,' + Buffer.from(mappingCode).toString('base64')
)
const liveCategories = await backend('/categories')
const products = []
for (let page = 1; ; page++) {
	const batch = await backend(`/products?pageSize=500&page=${page}`)
	products.push(...batch)
	if (batch.length < 500) break
}
const ids = new Set(liveCategories.map((c) => c.id))
const liveMenu = {
	categories: liveCategories.map(toCategory),
	dishes: products.filter((p) => ids.has(p.categoryId)).map(toDish),
}
const configResponse = await fetch(api + '/homepage-menu')
assert(configResponse.ok || configResponse.status === 404)
const original = configResponse.ok ? await configResponse.json() : { categories: [] }
const browser = await chromium.launch()
let changed = false
async function verify(configuration) {
	const groups = selectHomepageMenu(liveMenu, configuration, placeholder)
	assert(groups.length >= 2, 'At least two nonempty homepage categories required for switching QA')
	for (const width of [390, 1440])
		for (const theme of ['light', 'dark']) {
			const context = await browser.newContext({ viewport: { width, height: 1000 } })
			await context.addInitScript((theme) => localStorage.setItem('theme', theme), theme)
			const page = await context.newPage()
			const errors = []
			page.on('pageerror', (error) => errors.push(error.message))
			try {
				await page.goto(base + '/')
				const nav = page.getByRole('navigation', { name: 'Категории доставки на главной' })
				await expect(nav.getByRole('button')).toHaveText(groups.map((g) => g.category.name))
				for (const group of groups) {
					const button = nav.getByRole('button', { name: group.category.name, exact: true })
					await button.focus()
					const before = await page.evaluate(() => scrollY)
					await page.keyboard.press('Enter')
					await expect(button).toHaveAttribute('aria-pressed', 'true')
					await expect(button).toBeFocused()
					assert.equal(new URL(page.url()).pathname, '/')
					assert.equal(new URL(page.url()).search, '')
					assert(Math.abs((await page.evaluate(() => scrollY)) - before) <= 2, 'switching must preserve page scroll')
					const panel = page.getByRole('region', { name: group.category.name, exact: true })
					await expect(panel.locator('article strong')).toHaveText(group.dishes.map((d) => d.name))
					for (const item of group.dishes.filter((d) => d.orderBlocked)) {
						const card = panel.locator('article').filter({ has: page.getByText(item.name, { exact: true }) })
						await expect(card.getByRole('button')).toHaveCount(0)
					}
				}
				const cartGroup = groups.find((group) =>
					group.dishes.some((d) => d.available && !d.orderBlocked && d.variants[0].price !== null)
				)
				if (cartGroup) {
					await nav.getByRole('button', { name: cartGroup.category.name, exact: true }).click()
					const panel = page.getByRole('region', { name: cartGroup.category.name, exact: true })
					await panel
						.getByRole('button', { name: /^Добавить в корзину:/, disabled: false })
						.first()
						.click()
					const stored = await page.evaluate(() => localStorage.getItem('citypark-cart'))
					assert(stored)
					await page.reload()
					await expect(nav).toBeVisible()
					assert.equal(await page.evaluate(() => localStorage.getItem('citypark-cart')), stored)
				}
				assert(
					await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
					'no horizontal page overflow'
				)
				assert.deepEqual(errors, [])
				console.log(`PASS homepage switching, ordering, keyboard, scroll, cart and layout: ${width} ${theme}`)
			} finally {
				await context.close()
			}
		}
}
try {
	await verify(original)
	if (process.env.QA_HOMEPAGE_WRITES === '1') {
		assert(configResponse.ok, 'Deploy backend92 before curated QA')
		for (const selection of original.categories) {
			assert(ids.has(selection.categoryId), 'Repair stale original configuration before QA writes')
			for (const id of selection.productIds)
				assert(products.some((p) => p.id === id && p.categoryId === selection.categoryId))
		}
		token = (
			await backend(
				'/auth/sign-in',
				{ email: process.env.QA_ADMIN_EMAIL ?? 'dev@dev.io', password: process.env.QA_ADMIN_PASSWORD ?? 'devdevdev' },
				'POST'
			)
		).token
		const eligible = liveCategories
			.filter((c) => products.filter((p) => p.categoryId === c.id).length >= 3)
			.slice(0, 2)
			.reverse()
		assert.equal(eligible.length, 2)
		const curated = {
			categories: eligible.map((c) => ({
				categoryId: c.id,
				productIds: products
					.filter((p) => p.categoryId === c.id)
					.slice(0, 3)
					.reverse()
					.map((p) => p.id),
			})),
		}
		changed = true
		await backend('/homepage-menu', curated, 'PUT')
		await verify(curated)
		await backend('/homepage-menu', { categories: [] }, 'PUT')
		await verify({ categories: [] })
	} else console.log('Live writes not enabled; verified saved configuration or backend-404 automatic fallback')
} finally {
	if (changed) {
		await backend('/homepage-menu', original, 'PUT')
		console.log('Restored original homepage configuration')
	}
	await browser.close()
}
