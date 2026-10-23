import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

const LABELS = [
	'VIP-номер',
	'Одноместный с балконом',
	'Одноместный',
	'Двухместный, 2 кровати',
	'Двухместный, кровать евро',
]

test('shows the free room counts or the no-data message', async ({ page }) => {
	await signIn(page)
	await page.goto('/free-rooms')
	await expect(page.getByRole('heading', { name: 'Свободные номера' })).toBeVisible()

	const noData = page.getByText('Данных пока нет')
	const total = page.getByText(/Свободно: \d+ из 12/)
	await expect(noData.or(total)).toBeVisible()

	if (await total.isVisible()) {
		for (const label of LABELS) {
			await expect(page.getByRole('listitem').filter({ hasText: label }).first()).toBeVisible()
		}
		await expect(page.getByText(/Обновлено: /)).toBeVisible()
	}
})
