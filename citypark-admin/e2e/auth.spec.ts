import { expect, test } from '@playwright/test'
import { ADMIN_EMAIL, ADMIN_PASSWORD, signIn } from './helpers'

test('a wrong password shows the error and keeps the form filled', async ({ page }) => {
	await page.goto('/login')
	await page.getByLabel('E-mail').fill(ADMIN_EMAIL)
	await page.getByLabel('Пароль').fill('wrong-password')
	await page.getByRole('button', { name: 'Войти' }).click()
	await expect(page.getByText('Неверный e-mail или пароль')).toBeVisible()
	await expect(page.getByLabel('E-mail')).toHaveValue(ADMIN_EMAIL)
	await expect(page.getByLabel('Пароль')).toHaveValue('wrong-password')
	await expect(page).toHaveURL(/\/login/)
})

test('the dev admin lands on orders, keeps the session on reload and signs out', async ({ page }) => {
	await signIn(page)
	await expect(page).toHaveURL(/\/orders$/)
	await expect(page.getByRole('link', { name: 'Заказы' })).toHaveAttribute('aria-current', 'page')
	await expect(page.getByText(ADMIN_EMAIL)).toBeVisible()

	await page.reload()
	await expect(page.getByRole('heading', { name: 'Заказы' })).toBeVisible()

	await page.getByRole('button', { name: 'Выйти' }).click()
	await expect(page).toHaveURL(/\/login/)
	await page.goto('/orders')
	await expect(page).toHaveURL(/\/login/)
})

test('a signed-out visit goes to sign-in and comes back after it', async ({ page }) => {
	await page.goto('/delivery')
	await expect(page).toHaveURL(/\/login\?next=/)
	await page.getByLabel('E-mail').fill(ADMIN_EMAIL)
	await page.getByLabel('Пароль').fill(ADMIN_PASSWORD)
	await page.getByRole('button', { name: 'Войти' }).click()
	await expect(page).toHaveURL(/\/delivery$/)
	await expect(page.getByRole('heading', { name: 'Доставка' })).toBeVisible()
})

test('a signed-in admin is sent away from /login', async ({ page }) => {
	await signIn(page)
	await page.goto('/login')
	await expect(page).toHaveURL(/\/orders$/)
})

test('on a phone the menu button opens the drawer and navigates', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 800 })
	await signIn(page)
	await page.getByRole('button', { name: 'Открыть меню' }).click()
	await page.getByRole('dialog').getByRole('link', { name: 'Меню' }).click()
	await expect(page).toHaveURL(/\/menu$/)
	await expect(page.getByRole('heading', { name: 'Меню' })).toBeVisible()
	await expect(page.getByRole('dialog')).toBeHidden()
})
