import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

test('groups, search and the sync button', async ({ page }) => {
	await signIn(page)
	await page.getByRole('link', { name: 'Меню' }).first().click()
	await expect(page.getByRole('heading', { name: 'Меню' })).toBeVisible()
	await expect(page.getByText('Названия, цены, состав и фото меняются в iiko')).toBeVisible()

	const dishes = page.getByRole('list').last().getByRole('listitem')
	const groups = page.getByRole('navigation', { name: 'Категории меню' }).getByRole('button')
	await expect(groups.first()).toBeVisible()
	test.skip((await groups.count()) <= 1, 'the menu is empty in the dev database')

	const first = groups.nth(1)
	await first.click()
	await expect(first).toHaveAttribute('aria-pressed', 'true')
	await expect(page).toHaveURL(/category=\d+/)

	await groups.first().click()
	const name = (await dishes.first().locator('.font-medium').first().innerText()).trim()
	const search = page.getByLabel('Поиск блюда')
	await search.fill(name)
	await expect(dishes.filter({ hasText: name }).first()).toBeVisible()
	await expect(page).toHaveURL(/q=/)
	await search.fill(`нет-такого-${Date.now()}`)
	await expect(page.getByText('Блюд не найдено')).toBeVisible()

	const sync = page.getByRole('button', { name: 'Обновить меню из iiko' })
	await sync.click()
	// iiko may be unconfigured in dev: either toast is fine, but the button must come back.
	await expect(
		page.getByText(/Меню обновлено|Обновление уже идёт|Не удалось|Ошибка|Что-то пошло не так/).first()
	).toBeVisible({
		timeout: 30_000,
	})
	await expect(sync).toBeEnabled()
})
