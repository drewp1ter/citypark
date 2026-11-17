// Isolated UI regression with intercepted API responses; never writes to the backend.
import { expect, test } from '@playwright/test'
import type { components } from '../src/shared/api/schema'

function product(id: number, isHidden: boolean, isOrderBlocked = false): components['schemas']['Product'] {
	return {
		id,
		categoryId: 1,
		name: `Блюдо ${id}`,
		description: '',
		fullDescription: '',
		measureUnit: '',
		weight: 0,
		proteins: 0,
		fats: 0,
		carbohydrates: 0,
		calories: 0,
		flags: 0,
		price: 100,
		images: [],
		isHidden,
		isOrderBlocked,
		isAvailableForOrder: !isHidden && !isOrderBlocked,
		salesDays: '',
		startTime: '',
		endTime: '',
	}
}

for (const width of [390, 1440]) {
	for (const colorScheme of ['light', 'dark'] as const) {
		test(`staff can inspect hidden dishes at ${width}px in ${colorScheme}`, async ({ page }) => {
			await page.setViewportSize({ width, height: 900 })
			await page.emulateMedia({ colorScheme })
			await page.addInitScript(() => {
				sessionStorage.setItem(
					'citypark-admin-token',
					JSON.stringify({ token: 'visibility-test', email: 'test@example.com', name: 'Test' })
				)
			})
			await page.route('**/api/v1/**', (route) => {
				expect(route.request().method()).toBe('GET')
				const path = new URL(route.request().url()).pathname
				if (path.endsWith('/categories')) {
					return route.fulfill({
						json: [
							{
								id: 1,
								name: 'Кухня',
								description: '',
								images: [],
								isAvailableForOrder: true,
								salesDays: '',
								startTime: '',
								endTime: '',
							},
						],
					})
				}
				if (path.endsWith('/products'))
					return route.fulfill({ json: [product(1, false), product(2, true), product(3, true, true)] })
				return route.fulfill({ status: 503, json: { message: 'Unexpected request in isolated visibility test' } })
			})
			await page.goto('/menu')
			const visible = page.getByRole('listitem').filter({ hasText: 'Блюдо 1' })
			const hidden = page.getByRole('listitem').filter({ hasText: 'Блюдо 2' })
			const blocked = page.getByRole('listitem').filter({ hasText: 'Блюдо 3' })
			await expect(visible.getByText('Скрыто в iiko')).toHaveCount(0)
			await expect(hidden.getByText('Скрыто в iiko')).toBeVisible()
			await expect(hidden.getByText('Не в продаже сейчас')).toHaveCount(0)
			await expect(hidden.getByRole('switch', { name: 'Заблокировать заказ' })).toBeEnabled()
			await expect(blocked.getByText('Скрыто в iiko')).toBeVisible()
			await expect(blocked.getByText('Заказ заблокирован', { exact: true })).toBeVisible()
			await expect(blocked.getByRole('switch', { name: 'Заблокировать заказ' })).toBeChecked()
			await page.getByLabel('Поиск блюда').fill('Блюдо 2')
			await expect(hidden).toBeVisible()
			await hidden.getByRole('button', { name: 'Полное описание: Блюдо 2' }).click()
			await expect(page).toHaveURL(/dish=2/)
		})
	}
}
