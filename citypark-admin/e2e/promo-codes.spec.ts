import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

test.use({ timezoneId: 'America/Los_Angeles' })

for (const width of [390, 1440])
	for (const theme of ['light', 'dark']) {
		test(`campaign management ${width}px ${theme}`, async ({ page }) => {
			test.setTimeout(120000)
			await page.setViewportSize({ width, height: 900 })
			await page.addInitScript((value) => localStorage.setItem('citypark-admin-theme', value), theme)
			await signIn(page)
			await page.goto('/promo-codes')
			await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
			await expect(page.getByRole('heading', { name: 'Промокоды', exact: true })).toBeVisible()
			const codes = [
				`QA_${Date.now()}_${width}_${theme === 'light' ? 'L' : 'D'}`,
				`QS_${Date.now()}_${width}_${theme === 'light' ? 'L' : 'D'}`,
			] as const
			const created = new Set<string>()
			const editor = page.getByRole('region', { name: 'Редактор промокода' })
			async function save() {
				const response = page.waitForResponse(
					(r) => r.url().includes('/promo-codes') && ['POST', 'PUT'].includes(r.request().method())
				)
				await editor.getByRole('button', { name: 'Сохранить', exact: true }).click()
				const result = await response
				expect(result.ok()).toBe(true)
				const saved: { code: string } = await result.json()
				created.add(saved.code)
				await expect(editor).toHaveCount(0)
			}
			try {
				await page.getByRole('button', { name: 'Создать промокод' }).click()
				await editor.getByLabel('Промокод', { exact: true }).fill(codes[0])
				await editor.getByLabel('Процент скидки').fill('99')
				await expect(editor.getByLabel('Активен', { exact: true })).not.toBeChecked()
				await editor.getByLabel('Активен', { exact: true }).check()
				await save()
				await page.getByRole('button', { name: `Изменить ${codes[0]}`, exact: true }).click()
				await expect(editor.getByLabel('Промокод', { exact: true })).toHaveAttribute('readonly', '')
				await editor.getByLabel('Минимальная сумма подходящих блюд').fill('1000')
				await editor.getByLabel('Максимальная скидка').fill('250')
				await editor.getByLabel('Начало', { exact: true }).fill('2026-01-01T12:00')
				await editor.getByLabel('Окончание', { exact: true }).fill('2028-01-01T13:00')
				await save()
				await page.reload()
				await page.getByRole('button', { name: `Изменить ${codes[0]}`, exact: true }).click()
				await expect(editor.getByLabel('Минимальная сумма подходящих блюд')).toHaveValue('1000')
				await expect(editor.getByLabel('Начало', { exact: true })).toHaveValue('2026-01-01T12:00')
				await expect(editor.getByLabel('Окончание', { exact: true })).toHaveValue('2028-01-01T13:00')
				await editor.getByRole('button', { name: 'Отмена' }).click()

				await page.getByRole('button', { name: 'Создать промокод' }).click()
				await editor.getByLabel('Промокод', { exact: true }).fill(codes[0])
				await editor.getByRole('button', { name: 'Сохранить', exact: true }).click()
				await expect(page.getByText('Такой промокод уже существует.').first()).toBeVisible()
				await expect(editor.getByLabel('Промокод', { exact: true })).toHaveValue(codes[0])
				await editor.getByLabel('Промокод', { exact: true }).fill(codes[1])
				await editor.getByLabel('Процент скидки').fill('1')
				await editor.getByLabel('Выбранные блюда и категории', { exact: true }).check()
				const dishes = editor.getByRole('group', { name: /^Блюда/ })
				const categories = editor.getByRole('group', { name: /^Категории/ })
				await expect(dishes.getByRole('checkbox').first()).toBeVisible()
				const dish = dishes.getByRole('checkbox').first()
				const dishName = await dish.evaluate((el) => el.parentElement?.textContent?.trim() ?? '')
				await editor.getByLabel('Поиск: блюда').fill(dishName)
				await dishes.getByRole('checkbox').first().focus()
				await page.keyboard.press('Space')
				await expect(dishes.getByRole('checkbox').first()).toBeChecked()
				const categoryName = await categories
					.getByRole('checkbox')
					.first()
					.evaluate((el) => el.parentElement?.textContent?.trim() ?? '')
				await editor.getByLabel('Поиск: категории').fill(categoryName)
				await categories.getByRole('checkbox').first().check()
				await page.route('**/api/v1/promo-codes', (route) =>
					route.request().method() === 'POST'
						? route.fulfill({
								status: 400,
								contentType: 'application/json',
								body: JSON.stringify({ message: 'Проверьте кампанию', errors: { percentage: 'Проверьте процент' } }),
							})
						: route.continue()
				)
				await editor.getByRole('button', { name: 'Сохранить', exact: true }).click()
				await expect(editor.getByText('Проверьте процент', { exact: true })).toBeVisible()
				await expect(editor.getByLabel('Промокод', { exact: true })).toHaveValue(codes[1])
				await expect(dishes.getByRole('checkbox').first()).toBeChecked()
				await page.unroute('**/api/v1/promo-codes')
				await save()
				await page.getByRole('button', { name: `Изменить ${codes[1]}`, exact: true }).click()
				await expect(editor.getByLabel('Выбранные блюда и категории', { exact: true })).toBeChecked()
				await expect(
					editor.getByRole('group', { name: /^Блюда/ }).getByRole('checkbox', { checked: true })
				).toHaveCount(1)
				await expect(
					editor.getByRole('group', { name: /^Категории/ }).getByRole('checkbox', { checked: true })
				).toHaveCount(1)
				expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
			} finally {
				await page.unrouteAll({ behavior: 'wait' })
				await page.goto('/promo-codes')
				for (const code of created) {
					const edit = page.getByRole('button', { name: `Изменить ${code}`, exact: true })
					await expect(page.getByRole('heading', { name: 'Промокоды', exact: true })).toBeVisible()
					await expect(edit).toBeVisible()
					await edit.click()
					await editor.getByLabel('Активен', { exact: true }).uncheck()
					await save()
					await expect(
						page.getByRole('listitem').filter({ has: page.getByRole('heading', { name: code, exact: true }) })
					).toContainText('Отключён')
				}
			}
		})
	}
