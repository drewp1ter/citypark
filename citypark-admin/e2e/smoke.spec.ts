import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

test('opens on orders after sign-in and the theme survives a reload', async ({ page }) => {
	await page.emulateMedia({ colorScheme: 'light' })
	await signIn(page)
	await expect(page.getByRole('heading', { name: 'Заказы' })).toBeVisible()

	const html = page.locator('html')
	await expect(html).not.toHaveClass(/dark/)

	await page.getByRole('button', { name: 'Сменить тему' }).click()
	await expect(html).toHaveClass(/dark/)
	await expect(html).toHaveAttribute('data-theme', 'dark')

	await page.reload()
	await expect(html).toHaveClass(/dark/)

	await page.getByRole('button', { name: 'Сменить тему' }).click()
	await expect(html).not.toHaveClass(/dark/)
})
