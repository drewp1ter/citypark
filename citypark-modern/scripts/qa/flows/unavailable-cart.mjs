// Usage: node --env-file=.env scripts/qa/flows/unavailable-cart.mjs
import { chromium } from 'playwright'
import { expect } from 'playwright/test'
import { seedCart } from '../menu-cart.mjs'

const base = process.env.BASE_URL ?? 'http://localhost:3111'
const [available] = await seedCart([1])
const stale = { dishId: 'removed-qa-dish', variantId: 'default', quantity: 2 }
const browser = await chromium.launch()
const warning = 'Удалите недоступные блюда, чтобы продолжить.'
const removal = { name: 'Удалить из корзины: Недоступное блюдо' }
try {
	for (const width of [390, 1440]) {
		for (const theme of ['light', 'dark']) {
			const context = await browser.newContext({
				viewport: { width, height: 900 },
				storageState: {
					cookies: [],
					origins: [
						{
							origin: new URL(base).origin,
							localStorage: [
								{ name: 'theme', value: theme },
								{ name: 'citypark-cart', value: JSON.stringify([stale]) },
							],
						},
					],
				},
			})
			try {
				const page = await context.newPage()
				await page.goto(`${base}/menu`)
				await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
				const cart =
					width < 1024
						? page.getByRole('dialog', { name: 'Корзина' })
						: page.getByRole('complementary', { name: 'Корзина' })
				if (width < 1024) await page.getByRole('navigation', { name: 'Корзина' }).getByRole('button').press('Enter')
				await expect(cart.getByText(warning)).toBeVisible()
				await expect(cart.getByRole('button', { name: 'Оформить заказ' })).toBeDisabled()
				await cart.getByRole('button', removal).press('Enter')
				await expect(cart.getByText(/Пока пусто/)).toBeVisible()
				await page.reload()
				await expect(page.getByText(warning)).toHaveCount(0)
				await expect.poll(() => page.evaluate(() => localStorage.getItem('citypark-cart'))).toBeNull()

				await page.evaluate((line) => localStorage.setItem('citypark-cart', JSON.stringify([line])), stale)
				await page.goto(`${base}/checkout`)
				const summary = page.getByRole('complementary', { name: 'Ваш заказ' })
				const pay = page.getByRole('button', { name: 'Оплатить заказ' })
				await expect(summary.getByText(warning)).toBeVisible()
				await expect(pay).toBeDisabled()
				await summary.getByRole('button', removal).press('Enter')
				await expect(page.getByRole('heading', { name: 'В корзине пока пусто' })).toBeVisible()

				await page.evaluate((lines) => localStorage.setItem('citypark-cart', JSON.stringify(lines)), [available, stale])
				await page.goto(`${base}/checkout`)
				await page.getByText('Самовывоз', { exact: true }).click()
				await page.getByLabel('Имя').fill('Анна')
				await expect(summary.getByText(warning)).toBeVisible()
				await expect(pay).toBeDisabled()
				await summary.getByRole('button', removal).press('Enter')
				await expect(summary.getByText(warning)).toHaveCount(0)
				await expect(pay).toBeEnabled()
				await expect(page.getByLabel('Имя')).toHaveValue('Анна')
				await expect(summary.getByLabel('Расчёт заказа').getByText(/₽/).first()).toBeVisible()
				await page.reload()
				await expect
					.poll(() => page.evaluate(() => localStorage.getItem('citypark-cart')))
					.toBe(JSON.stringify([available]))
				await expect(summary.getByText(warning)).toHaveCount(0)
				console.log(`PASS unavailable cart, keyboard removal, re-quote and reload at ${width}px (${theme})`)
			} finally {
				await context.close()
			}
		}
	}
} finally {
	await browser.close()
}
