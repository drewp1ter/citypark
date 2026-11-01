import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

for (const width of [390, 1440])
	for (const theme of ['light', 'dark'] as const) {
		test(`delivery information ${width} ${theme}`, async ({ page }) => {
			await page.setViewportSize({ width, height: 1000 })
			await page.addInitScript((theme) => localStorage.setItem('citypark-admin-theme', theme), theme)
			await signIn(page)
			await page.goto('/delivery')
			await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
			const card = page.getByRole('region', { name: 'Информация о доставке' })
			const save = card.getByRole('button', { name: 'Сохранить' })
			const labels = ['Время приготовления', 'Зона доставки', 'Цена от, ₽', 'Цена до, ₽'] as const
			await expect(card).toBeVisible()
			await expect(save).toBeDisabled()
			const original = await Promise.all(labels.map((label) => card.getByLabel(label, { exact: true }).inputValue()))
			async function fill(values: string[]) {
				for (const [i, label] of labels.entries()) await card.getByLabel(label, { exact: true }).fill(values[i] ?? '')
			}
			async function persist() {
				const response = page.waitForResponse(
					(r) => r.request().method() === 'PUT' && r.url().includes('/delivery-settings')
				)
				await save.click()
				expect((await response).ok()).toBe(true)
				await expect(save).toBeDisabled()
			}
			try {
				await fill(['Тест 35 мин', 'Тест СОТ', '210', '360'])
				await persist()
				await page.reload()
				await expect(card.getByLabel(labels[0], { exact: true })).toHaveValue('Тест 35 мин')
				await card.getByLabel(labels[0], { exact: true }).fill(' ')
				await save.click()
				await expect(card.getByText('поле не должно быть пустым')).toBeVisible()
				await fill(['Черновик', 'Тест СОТ', '400', '300'])
				await save.click()
				await expect(card.getByText('Максимальная цена должна быть не меньше минимальной')).toBeVisible()
				await fill(['Черновик', 'Тест СОТ', '210', '360'])
				// Hold a failing request to verify pending state and transport recovery.
				let release!: () => void
				const gate = new Promise<void>((resolve) => {
					release = resolve
				})
				await page.route('**/delivery-settings', async (route) => {
					if (route.request().method() === 'PUT') {
						await gate
						await route.abort()
					} else await route.continue()
				})
				await save.click()
				await expect(save).toBeDisabled()
				release()
				await expect(save).toBeEnabled()
				await expect(card.getByLabel(labels[0], { exact: true })).toHaveValue('Черновик')
				await page.unroute('**/delivery-settings')

				await page.route('**/delivery-settings', async (route) => {
					if (route.request().method() === 'PUT')
						await route.fulfill({
							status: 400,
							json: { statusCode: 400, message: 'Ошибка проверки', errors: { cooking: 'Ошибка поля' } },
						})
					else await route.continue()
				})
				await save.click()
				await expect(card.getByText('Ошибка поля')).toBeVisible()
				await expect(card.getByLabel(labels[0], { exact: true })).toHaveValue('Черновик')
				// A real background fetch must not erase unsaved input.

				const refreshed = page.waitForResponse(
					(r) => r.request().method() === 'GET' && r.url().includes('/delivery-settings')
				)
				await page.evaluate(() => window.dispatchEvent(new Event('visibilitychange')))
				await refreshed

				await expect(card.getByLabel(labels[0], { exact: true })).toHaveValue('Черновик')
				await page.unroute('**/delivery-settings')
				await expect(page.getByRole('heading', { name: 'Часы приёма заказов' })).toBeVisible()
				await expect(page.getByLabel('Поиск города')).toBeVisible()
				await page.getByLabel('Поиск города').fill('Несуществующий город')
				await expect(page.getByText('Ничего не найдено')).toBeVisible()
				await page.getByLabel('Поиск города').fill('')
				await page.getByRole('button', { name: 'Добавить город' }).click()
				await expect(page.getByRole('dialog')).toBeVisible()
				await page.keyboard.press('Escape')
			} finally {
				await page.unroute('**/delivery-settings')
				await page.goto('/delivery')
				await expect(card).toBeVisible()
				await fill(original)
				if (await save.isEnabled()) await persist()
			}
		})
	}

test('loading failure retries independently of town list', async ({ page }) => {
	await signIn(page)
	await page.route('**/delivery-settings', (route) => route.abort())
	await page.goto('/delivery')
	await expect(page.getByLabel('Загрузка информации о доставке')).toBeVisible()
	await expect(page.getByRole('button', { name: 'Повторить', exact: true })).toBeVisible({ timeout: 20000 })
	await expect(page.getByLabel('Поиск города')).toBeVisible()
	await page.unroute('**/delivery-settings')
	await page.getByRole('button', { name: 'Повторить', exact: true }).click()
	await expect(page.getByRole('region', { name: 'Информация о доставке' })).toBeVisible()
})
