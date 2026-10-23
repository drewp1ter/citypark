import { expect, test } from '@playwright/test'

test('shows the heading and the theme survives a reload', async ({ page }) => {
	await page.emulateMedia({ colorScheme: 'light' })
	await page.goto('/')
	await expect(page.getByRole('heading', { name: 'Сити Парк — админка' })).toBeVisible()

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
