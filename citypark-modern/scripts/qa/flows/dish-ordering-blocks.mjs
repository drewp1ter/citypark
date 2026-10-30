// Read-only flow. Set QA_BLOCKED_DISH_ID to a dish already blocked through admin.
// No backend writes or orders are submitted by this flow.
import assert from 'node:assert/strict'
import { chromium } from 'playwright'
const base = process.env.BASE_URL ?? 'http://localhost:3111'
const api = process.env.MENU_API_URL
if (!api) throw new Error('Set MENU_API_URL')
const get = async (path) => {
	const response = await fetch(api + path)
	assert(response.ok, `GET ${path}: ${response.status}`)
	return response.json()
}
const visible = new Set((await get('/categories')).map((c) => c.id))
const products = await get('/products?pageSize=500')
const dish = products.find(
	(p) =>
		visible.has(p.categoryId) &&
		p.isOrderBlocked &&
		(!process.env.QA_BLOCKED_DISH_ID || String(p.id) === process.env.QA_BLOCKED_DISH_ID)
)
if (!dish) {
	if (process.env.QA_REQUIRE_BLOCKED_DISH === '1' || process.env.QA_BLOCKED_DISH_ID)
		throw new Error('Block a visible dish through admin before running this flow')
	console.log(
		'Blocked-dish checks need a fixture: run admin menu e2e with QA_PUBLIC_BASE_URL, or block a dish in admin and rerun.'
	)
	process.exit(0)
}
const browser = await chromium.launch()
try {
	for (const width of [390, 1440])
		for (const colorScheme of ['light', 'dark']) {
			const context = await browser.newContext({ viewport: { width, height: 900 }, colorScheme })
			await context.addInitScript(
				(id) =>
					localStorage.setItem('citypark-cart', JSON.stringify([{ dishId: id, variantId: 'default', quantity: 2 }])),
				String(dish.id)
			)
			const page = await context.newPage()
			for (const path of ['/menu']) {
				await page.goto(`${base}${path}?category=${dish.categoryId}`)
				const card = page
					.locator('article')
					.filter({ has: page.getByRole('heading', { name: dish.name, exact: true }) })
					.first()
				await card.waitFor()
				assert.equal(await card.getByRole('button', { name: /Добавить/ }).count(), 0)
				assert.equal(await card.getByText(/Временно недоступно|Заказ заблокирован/).count(), 0)
			}
			await page.goto(base + '/checkout')
			if (width < 768) await page.locator('details summary').first().click()
			await page
				.getByRole('button', { name: `Удалить из корзины: ${dish.name}` })
				.filter({ visible: true })
				.first()
				.click()
			await page.getByRole('heading', { name: 'В корзине пока пусто' }).waitFor()
			await context.close()
		}
	console.log('Blocked dishes: visible without purchase controls; persisted cart removal passes at both sizes/themes.')
} finally {
	await browser.close()
}
