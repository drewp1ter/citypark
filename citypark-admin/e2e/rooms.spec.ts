import { expect, test, type Page } from '@playwright/test'
import { signIn } from './helpers'

function singleCard(page: Page) {
	return page.getByRole('listitem').filter({ has: page.getByRole('heading', { name: 'Одноместный', exact: true }) })
}

test('edits a room price and description, and asks before leaving with changes', async ({ page }) => {
	await signIn(page)
	await page.goto('/content/rooms')
	await expect(page.getByRole('heading', { name: 'Номера' })).toBeVisible()
	await expect(page.getByRole('link', { name: /^Изменить/ })).toHaveCount(5)

	const card = singleCard(page)
	await card.getByRole('link', { name: /Изменить/ }).click()
	await expect(page.getByRole('link', { name: '← Номера' })).toBeVisible()

	const price = page.getByLabel('Цена за ночь, ₽')
	const original = await price.inputValue()
	const description = page.getByRole('textbox', { name: 'Описание', exact: true })
	const originalText = (await description.textContent()) ?? ''
	const testPrice = '4321'

	try {
		await price.fill(testPrice)
		await description.click()
		await page.keyboard.press('Control+A')
		await page.keyboard.type('Тест жирный')
		await page.keyboard.press('Shift+Control+ArrowLeft')
		await page.getByRole('button', { name: 'Жирный' }).click()
		await page.getByRole('button', { name: 'Сохранить' }).click()
		await expect(page.getByText('Сохранено')).toBeVisible()

		// The list shows the new price.
		await page.getByRole('link', { name: '← Номера' }).click()
		await expect(singleCard(page)).toContainText('4 321 ₽')

		// Reload keeps the price and the bold word.
		await singleCard(page)
			.getByRole('link', { name: /Изменить/ })
			.click()
		await page.reload()
		await expect(page.getByLabel('Цена за ночь, ₽')).toHaveValue(testPrice)
		await expect(page.getByRole('textbox', { name: 'Описание', exact: true }).locator('strong')).toHaveCount(1)

		// Leaving with a change asks first; dismissing the dialog keeps us on the page.
		await page.getByLabel('Строка на странице отеля').fill(`Тест ${Date.now()}`)
		let message = ''
		page.once('dialog', (dialog) => {
			message = dialog.message()
			void dialog.dismiss()
		})
		await page.getByRole('link', { name: '← Номера' }).click()
		await expect.poll(() => message).toBe('Есть несохранённые изменения. Уйти без сохранения?')
		await expect(page.getByLabel('Цена за ночь, ₽')).toBeVisible()
	} finally {
		await page.goto('/content/rooms')
		await singleCard(page)
			.getByRole('link', { name: /Изменить/ })
			.click()
		await page.getByLabel('Цена за ночь, ₽').fill(original)
		// Put the description back (the test only ever starts from an empty or plain one).
		await page.getByRole('textbox', { name: 'Описание', exact: true }).click()
		await page.keyboard.press('Control+A')
		if (originalText === '') await page.keyboard.press('Delete')
		else await page.keyboard.type(originalText)
		const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/rooms/'))
		await page.getByRole('button', { name: 'Сохранить' }).click()
		const res = await saved
		expect(res.ok(), await res.text()).toBe(true)
	}
})
