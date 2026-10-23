import { expect, type Page } from '@playwright/test'

export const ADMIN_EMAIL = 'dev@dev.io'
export const ADMIN_PASSWORD = 'devdevdev'

/** Signs in through the UI with the dev admin and waits for the first page. */
export async function signIn(page: Page): Promise<void> {
	await page.goto('/login')
	await page.getByLabel('E-mail').fill(ADMIN_EMAIL)
	await page.getByLabel('Пароль').fill(ADMIN_PASSWORD)
	await page.getByRole('button', { name: 'Войти' }).click()
	await expect(page.getByRole('heading', { name: 'Заказы' })).toBeVisible()
}
