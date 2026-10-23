import { expect, test, type Page } from '@playwright/test'
import { moscowToday } from '../src/shared/lib/today'
import { shiftDay } from '../src/features/reports/model/periods'
import { signIn } from './helpers'

const SALES = {
	rows: [
		{ organization: 'ИП Иванов', room: 'Стандарт', count: 2, sum: 4000 },
		{ organization: 'ООО «Олимп»', room: 'VIP-номер', count: 3, sum: 12000.5 },
		{ organization: 'ООО «Олимп»', room: 'Люкс', count: 1, sum: 9000 },
	],
	total: { count: 6, sum: 25000.5 },
}

async function mockReports(page: Page) {
	await page.route('**/reports/hotel/daily**', async (route) => {
		const url = new URL(route.request().url())
		const from = url.searchParams.get('from') ?? ''
		const to = url.searchParams.get('to') ?? ''
		const days = []
		for (let day = from; day <= to; day = shiftDay(day, 1)) days.push({ date: day, sum: 1000 })
		await route.fulfill({ json: { days, total: days.length * 1000 } })
	})
	await page.route('**/reports/hotel/sales**', (route) => route.fulfill({ json: SALES }))
}

test('the dashboard shows the chart or says that 1C is not configured', async ({ page }) => {
	await signIn(page)
	await page.goto('/reports')
	await expect(page.getByRole('heading', { name: 'Отчёты', level: 1 })).toBeVisible()
	const notConfigured = page.getByText('Отчёты не настроены на сервере')
	const unavailable = page.getByText('1С не отвечает. Попробуйте позже.')
	const chart = page.getByRole('img', { name: 'График выручки по дням' })
	await expect(notConfigured.or(unavailable).or(chart)).toBeVisible({ timeout: 40_000 })
})

test('the sales report page has the period picker', async ({ page }) => {
	await signIn(page)
	await page.goto('/reports/hotel-sales')
	await expect(page.getByRole('heading', { name: 'Сдача номеров' })).toBeVisible()
	await expect(page.getByRole('button', { name: 'Этот месяц' })).toBeVisible()
	await expect(page.getByRole('group', { name: 'Быстрый выбор периода' })).toBeVisible()
})

test('an unknown report is not found', async ({ page }) => {
	await signIn(page)
	await page.goto('/reports/nope')
	await expect(page.getByText('Отчёт не найден')).toBeVisible()
})

test('the dashboard tiles and chart show the mocked days', async ({ page }) => {
	await mockReports(page)
	await signIn(page)
	const today = moscowToday()
	await page.goto(`/reports?from=${shiftDay(today, -9)}&to=${today}`)
	const tiles = page.getByRole('listitem').filter({ hasText: /Сегодня|Вчера|7 дней|За период/ })
	await expect(tiles.filter({ hasText: 'Сегодня' })).toContainText(/1\s000\s₽/)
	await expect(tiles.filter({ hasText: 'Вчера' })).toContainText(/1\s000\s₽/)
	await expect(tiles.filter({ hasText: '7 дней' })).toContainText(/7\s000\s₽/)
	await expect(tiles.filter({ hasText: 'За период' })).toContainText(/10\s000\s₽/)
	await expect(page.getByRole('img', { name: 'График выручки по дням' })).toBeVisible()
	await page.getByRole('link', { name: 'Сдача номеров' }).click()
	await expect(page).toHaveURL(/\/reports\/hotel-sales\?/)
})

test('the tile for a day outside the period is a dash', async ({ page }) => {
	await mockReports(page)
	await signIn(page)
	const today = moscowToday()
	await page.goto(`/reports?from=${shiftDay(today, -9)}&to=${shiftDay(today, -5)}`)
	await expect(page.getByRole('listitem').filter({ hasText: 'Сегодня' })).toContainText('—')
	await expect(page.getByRole('listitem').filter({ hasText: '7 дней' })).toContainText('—')
})

test('the tree is an accordion at 390 px', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 800 })
	await mockReports(page)
	await signIn(page)
	await page.goto('/reports/hotel-sales')
	const olymp = page.getByRole('button', { name: /ООО «Олимп»/ })
	await expect(olymp).toContainText(/21\s000,50\s₽/)
	await expect(page.getByText('VIP-номер')).toHaveCount(0)
	await olymp.click()
	await expect(page.getByText('VIP-номер')).toBeVisible()
	await expect(page.getByText('Люкс')).toBeVisible()
	await expect(page.getByText('Итого', { exact: true }).locator('..')).toContainText(/25\s000,50\s₽/)
})

test('the tree is a table with expandable rows at 1440 px', async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 900 })
	await mockReports(page)
	await signIn(page)
	await page.goto('/reports/hotel-sales')
	const table = page.getByRole('table', { name: 'Сдача номеров' })
	await expect(table.getByRole('row', { name: /ООО «Олимп»/ })).toContainText(/21\s000,50\s₽/)
	await expect(table.getByText('VIP-номер')).toHaveCount(0)
	await table.getByRole('button', { name: /ООО «Олимп»/ }).click()
	await expect(table.getByRole('row', { name: /VIP-номер/ })).toContainText(/12\s000,50\s₽/)
	await expect(table.getByRole('row', { name: /Итого/ })).toContainText(/25\s000,50\s₽/)
})

test('a 1C failure shows the message and a retry button', async ({ page }) => {
	await page.route('**/reports/hotel/sales**', (route) =>
		route.fulfill({ status: 502, json: { statusCode: 502, message: 'Сервис отчётов недоступен' } })
	)
	await signIn(page)
	await page.goto('/reports/hotel-sales')
	await expect(page.getByText('1С не отвечает. Попробуйте позже.')).toBeVisible()
	await expect(page.getByRole('button', { name: 'Повторить' })).toBeVisible()
})
