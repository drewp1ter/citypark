import assert from 'node:assert/strict'
import { gunzipSync } from 'node:zlib'
import { chromium, expect } from 'playwright/test'
import { seedCart } from './menu-cart.mjs'
const base = process.env.BASE_URL || 'http://localhost:3111'
const cart = await seedCart([1])
const browser = await chromium.launch()
try {
	for (const scenario of [
		'permission_denied',
		'position_unavailable',
		'location_timeout',
		'unsupported',
		'request_failed',
		'request_timeout',
		'invalid_response',
		'server_response',
		'manual_takeover',
		'unmounted',
	]) {
		const context = await browser.newContext()
		const events = []
		await context.route(/\/envelope\/?(?:\?|$)/, async (route) => {
			const buffer = route.request().postDataBuffer()
			const body = buffer?.[0] === 31 && buffer?.[1] === 139 ? gunzipSync(buffer).toString() : buffer?.toString() || ''
			for (const line of body.split('\n')) {
				try {
					const event = JSON.parse(line)
					if (event.tags?.autofill_reason) events.push(event)
				} catch {}
			}
			if (process.env.SEND_AUTOFILL_VERIFICATION === '1' && scenario === 'permission_denied') await route.continue()
			else await route.fulfill({ status: 200, body: '{}' })
		})
		await context.addInitScript(
			({ cart, scenario }) => {
				localStorage.setItem('citypark-cart', JSON.stringify(cart))
				const code = scenario === 'permission_denied' ? 1 : scenario === 'position_unavailable' ? 2 : 3
				Object.defineProperty(navigator, 'geolocation', {
					value:
						scenario === 'unsupported'
							? undefined
							: {
									getCurrentPosition: (success, failure) =>
										setTimeout(
											() => {
												if (['permission_denied', 'position_unavailable', 'location_timeout'].includes(scenario))
													failure({ code })
												else success({ coords: { latitude: 44, longitude: 39 } })
											},
											['manual_takeover', 'unmounted'].includes(scenario) ? 1500 : 0
										),
								},
				})
			},
			{ cart, scenario }
		)
		await context.route('**/api/address-suggest?**', (route) => route.fulfill({ json: [] }))
		const page = await context.newPage()
		await page.route('**/api/address-geolocate', async (route) => {
			if (scenario === 'request_failed') return route.abort()
			if (scenario === 'request_timeout') {
				await new Promise((resolve) => setTimeout(resolve, 6000))
				return route.fulfill({ json: { status: 'unmatched' } }).catch(() => {})
			}
			return route.fulfill({
				body: scenario === 'invalid_response' ? '{' : JSON.stringify({ status: 'unmatched' }),
				contentType: 'application/json',
			})
		})
		await page.goto(base + '/checkout')
		if (scenario === 'manual_takeover')
			await page.getByRole('combobox', { name: 'Улица и дом' }).fill('Synthetic manual address')
		if (['manual_takeover', 'unmounted', 'server_response'].includes(scenario)) {
			await page.waitForTimeout(2000)
			assert.equal(events.length, 0, scenario)
		} else {
			await page.waitForTimeout(2000)
			await expect.poll(() => events.length, { timeout: 15000 }).toBe(1)
			assert.equal(events[0].tags.autofill_reason, scenario)
			assert.equal(events[0].contexts.autofill.stage, 'browser')
			assert.equal(events[0].contexts.autofill.surface, 'web')
			assert.ok(events[0].exception.values[0].stacktrace.frames.length)
			assert.equal(JSON.stringify(events[0]).includes('latitude'), false)
		}
		await context.close()
		console.log(`Autofill browser ${scenario}: PASS`)
	}
} finally {
	await browser.close()
}
