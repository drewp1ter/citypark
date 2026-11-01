// Read-only production flow. Usage: MENU_API_URL=... node scripts/qa/flows/cart-motion.mjs
import assert from 'node:assert/strict'
import { chromium } from 'playwright'
import { seedCart } from '../menu-cart.mjs'

const base = process.env.BASE_URL ?? 'http://localhost:3111'
const cart = await seedCart([1])
const browser = await chromium.launch()
try {
	for (const theme of ['light', 'dark']) {
		for (const reducedMotion of ['no-preference', 'reduce']) {
			const context = await browser.newContext({ viewport: { width: 390, height: 844 }, reducedMotion, hasTouch: true })
			await context.addInitScript(
				({ cart, theme }) => {
					localStorage.setItem('citypark-cart', JSON.stringify(cart))
					localStorage.setItem('theme', theme)
				},
				{ cart, theme }
			)
			const page = await context.newPage()
			const errors = []
			page.on('pageerror', (error) => errors.push(error.message))
			await page.goto(base + '/menu')
			const bar = page.locator("nav[aria-label='Корзина'] button")
			const sheet = page.locator('dialog.cart-sheet')
			const moving = reducedMotion === 'no-preference'
			const open = async () => {
				await bar.click()
				assert.equal(await sheet.evaluate((e) => e.open), true)
				if (moving) {
					assert.ok(
						await sheet.evaluate((e) =>
							e.getAnimations().some((a) => a instanceof CSSTransition && a.transitionProperty === 'translate')
						),
						'opening slide exists'
					)
				}
				await page.waitForTimeout(320)
				assert.equal(await sheet.evaluate((e) => getComputedStyle(e).opacity), '1')
			}
			const closed = async () => {
				assert.equal(await sheet.evaluate((e) => e.open), false)
				if (moving) {
					assert.equal(
						await sheet.evaluate((e) => getComputedStyle(e).display),
						'block',
						'closing sheet stays rendered'
					)
					assert.ok(
						await sheet.evaluate((e) =>
							e.getAnimations().some((a) => a instanceof CSSTransition && a.transitionProperty === 'translate')
						),
						'closing slide exists'
					)
				}
				await page.waitForTimeout(320)
				assert.equal(await sheet.evaluate((e) => getComputedStyle(e).display), 'none')
				assert.equal(await bar.getAttribute('aria-expanded'), 'false')
				assert.equal(await bar.evaluate((e) => e === document.activeElement), true)
			}
			await open()
			await sheet.getByRole('button', { name: 'Закрыть корзину' }).click()
			await closed()
			await open()
			await page.keyboard.press('Escape')
			await closed()
			await open()
			await page.mouse.click(10, 10)
			await closed()
			await open()
			await sheet.getByRole('link', { name: 'Оформить заказ', exact: true }).click()
			await page.waitForURL('**/checkout')
			assert.deepEqual(errors, [])
			console.log(
				`PASS mobile cart: ${theme}, ${reducedMotion}; close button, Escape, backdrop, reopen, focus and checkout`
			)
			await context.close()
		}
	}
} finally {
	await browser.close()
}
