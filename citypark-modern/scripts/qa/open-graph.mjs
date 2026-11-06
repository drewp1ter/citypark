import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const origin = process.env.BASE_URL || 'http://localhost:3111'
const browser = await chromium.launch()
try {
	const page = await browser.newPage({ userAgent: 'TelegramBot (like TwitterBot)' })
	const paths = [
		'/',
		'/restaurant',
		'/hotel',
		'/rooms',
		'/menu?category=example',
		'/news',
		'/contacts',
		'/documents',
		'/privacy',
		'/offer',
	]
	for (const section of ['/rooms', '/news']) {
		await page.goto(origin + section)
		const href = await page
			.locator(`a[href^="${section}/"]`)
			.first()
			.getAttribute('href')
			.catch(() => null)
		assert.ok(href, `Missing detail link on ${section}`)
		paths.push(href)
	}
	for (const path of paths) {
		await page.goto(origin + path)
		const meta = async (key) => page.locator(`meta[property="${key}"], meta[name="${key}"]`).getAttribute('content')
		assert.equal(await meta('og:title'), await page.title())
		assert.ok(await meta('og:description'))
		assert.equal(await meta('og:locale'), 'ru_RU')
		assert.equal(await meta('twitter:card'), 'summary_large_image')
		assert.equal(await meta('twitter:title'), await meta('og:title'))
		assert.equal(await meta('twitter:description'), await meta('og:description'))
		assert.equal(await meta('og:url'), await page.locator('link[rel="canonical"]').getAttribute('href'))
		const image = await meta('og:image')
		assert.match(image, /^https:\/\//)
		const url = new URL(image)
		const response = await page.request.get(url.origin === 'https://cityparkvip.ru' ? origin + url.pathname : image)
		assert.ok(response.ok(), image)
		assert.ok(response.headers()['content-type'].startsWith('image/'))
		console.log('PASS', path)
	}
} finally {
	await browser.close()
}
