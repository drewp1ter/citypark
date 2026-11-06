import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const base = process.env.BASE_URL || 'http://localhost:3111'
const browser = await chromium.launch({ headless: true })
try {
	const page = await browser.newPage()
	await page.goto(`${base}/menu`)
	const href = await page.locator('h3 a[href^="/menu/"]').first().getAttribute('href')
	assert.ok(href, 'public menu links to dishes')
	const response = await fetch(base + href, { headers: { 'User-Agent': 'Twitterbot/1.0' } })
	assert.equal(response.status, 200)
	const html = await response.text()
	assert.ok(html.includes(`https://cityparkvip.ru${href}`))
	assert.ok(html.includes('property="og:image"'))
	assert.ok(html.includes('name="twitter:card"'))
	assert.ok((await (await fetch(`${base}/sitemap.xml`)).text()).includes(`https://cityparkvip.ru${href}`))
	assert.equal((await fetch(`${base}/menu/no-such-dish`, { headers: { 'User-Agent': 'Twitterbot/1.0' } })).status, 404)
	for (const width of [390, 1440])
		for (const colorScheme of ['light', 'dark']) {
			await page.setViewportSize({ width, height: 900 })
			await page.emulateMedia({ colorScheme })
			await page.goto(base + href)
			await page.locator('h1').waitFor()
			assert.ok(await page.locator('h1').textContent())
			assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
		}
	await page.evaluate(() =>
		Object.defineProperty(navigator, 'clipboard', {
			configurable: true,
			value: {
				writeText: async (text) => {
					window.copiedDishUrl = text
				},
			},
		})
	)
	await page.getByRole('button', { name: 'Копировать ссылку', exact: true }).click()
	await page.getByRole('status').filter({ hasText: 'Ссылка скопирована' }).waitFor()
	assert.equal(await page.evaluate(() => window.copiedDishUrl), `https://cityparkvip.ru${href}`)
	await page.evaluate(() =>
		Object.defineProperty(navigator, 'clipboard', {
			configurable: true,
			value: {
				writeText: async () => {
					throw new Error('denied')
				},
			},
		})
	)
	await page.getByRole('button', { name: 'Копировать ссылку', exact: true }).click()
	assert.equal(
		await page.getByRole('textbox', { name: 'Ссылка на блюдо' }).inputValue(),
		`https://cityparkvip.ru${href}`
	)
	await page.reload()
	await page.locator('h1').waitFor()
	await page.getByRole('link', { name: 'Меню', exact: true }).click()
	await page.waitForURL('**/menu?category=*')
	console.log(
		'PASS: dish navigation, crawler metadata, sitemap, 404, themes/viewports, clipboard success/failure, refresh and category return'
	)
} finally {
	await browser.close()
}
