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
