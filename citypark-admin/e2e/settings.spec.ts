import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

test('edits the e-mail, keeps it after a reload, and rejects a bad postcode', async ({ page }) => {
	await signIn(page)
	await page.goto('/content/settings')
	await expect(page.getByRole('heading', { name: 'Настройки сайта' })).toBeVisible()

	const email = page.getByLabel('E-mail', { exact: true })
	const postcode = page.getByLabel('Индекс')
	const save = page.getByRole('button', { name: 'Сохранить' })
	await expect(save).toBeDisabled()
	const original = await email.inputValue()
	const originalPostcode = await postcode.inputValue()
	const testEmail = `test${Date.now()}@example.com`

	try {
		await email.fill(testEmail)
		await expect(save).toBeEnabled()
		await save.click()
		await expect(page.getByText('Сохранено')).toBeVisible()

		await page.reload()
		await expect(page.getByLabel('E-mail', { exact: true })).toHaveValue(testEmail)

		// A 5-digit postcode: the error is under the field and nothing is saved.
		await page.getByLabel('Индекс').fill('35263')
		await expect(page.getByTestId('address-preview')).toContainText('35263,')
		await page.getByLabel('E-mail', { exact: true }).fill(`other${Date.now()}@example.com`)
		await save.click()
		await expect(page.getByText('Индекс — ровно 6 цифр')).toBeVisible()
		await page.reload()
		await expect(page.getByLabel('Индекс')).toHaveValue(originalPostcode)
		await expect(page.getByLabel('E-mail', { exact: true })).toHaveValue(testEmail)
	} finally {
		await page.goto('/content/settings')
		await page.getByLabel('E-mail', { exact: true }).fill(original)
		const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/settings'))
		await page.getByRole('button', { name: 'Сохранить' }).click()
		const res = await saved
		expect(res.ok(), await res.text()).toBe(true)
	}
})

test('fills the call number from the display text', async ({ page }) => {
	await signIn(page)
	await page.goto('/content/settings')
	const group = page.getByRole('group', { name: 'Бесплатно по России' })
	await group.getByLabel('Номер для звонка').first().fill('')
	await group.getByRole('button', { name: 'Подставить номер из отображения' }).first().click()
	await expect(group.getByLabel('Номер для звонка').first()).toHaveValue('88001002441')
	// Nothing is saved: leaving the page asks first, and the test just discards the page.
})

test('saves the promo switch, protects unsaved changes and preserves it on failure', async ({ page }) => {
	await signIn(page)
	await page.goto('/content/settings')
	const toggle = page.getByRole('switch', { name: 'Промокоды включены' })
	const save = page.getByRole('button', { name: 'Сохранить' })
	await expect(toggle).toBeVisible()
	const original = await toggle.isChecked()
	const saveAndReload = async (enabled: boolean) => {
		if ((await toggle.isChecked()) !== enabled) {
			await page.getByText('Промокоды включены', { exact: true }).click()
			const response = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/settings'))
			await save.click()
			expect((await response).ok()).toBe(true)
		}
		await page.reload()
		if (enabled) await expect(toggle).toBeChecked()
		else await expect(toggle).not.toBeChecked()
	}
	try {
		await saveAndReload(false)
		await page.getByText('Промокоды включены', { exact: true }).click()
		await expect(save).toBeEnabled()
		page.once('dialog', (dialog) => void dialog.dismiss())
		await page.getByRole('link', { name: 'Заказы', exact: true }).first().click()
		await expect(toggle).toBeChecked()
		await page.route('**/settings', async (route) => {
			if (route.request().method() === 'PUT') {
				await route.fulfill({
					status: 500,
					contentType: 'application/json',
					body: JSON.stringify({ message: 'Ошибка сервера' }),
				})
			} else await route.continue()
		})
		await save.click()
		await expect(page.getByText('Ошибка сервера')).toBeVisible()
		await expect(toggle).toBeChecked()
		await expect(save).toBeEnabled()
		await page.unroute('**/settings')
		const response = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/settings'))
		await save.click()
		expect((await response).ok()).toBe(true)
		await saveAndReload(true)
		await saveAndReload(false)
	} finally {
		await page.unroute('**/settings')
		page.once('dialog', (dialog) => void dialog.accept())
		await page.reload()
		await saveAndReload(original)
	}
})
