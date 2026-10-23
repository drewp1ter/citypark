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

test('edits the short text of the VIP room and keeps it after a reload', async ({ page }) => {
	await signIn(page)
	await page.goto('/content/rooms')
	const open = () =>
		page
			.getByRole('listitem')
			.filter({ has: page.getByRole('heading', { name: 'VIP-номер', exact: true }) })
			.getByRole('link', { name: /Изменить/ })
			.click()
	await open()
	const summary = page.getByLabel('Кратко', { exact: true })
	const original = await summary.inputValue()
	const text = `Тест ${Date.now()}`

	async function save() {
		const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/rooms/'))
		await page.getByRole('button', { name: 'Сохранить' }).click()
		const res = await saved
		expect(res.ok(), await res.text()).toBe(true)
	}

	try {
		await summary.fill(text)
		await save()
		await page.reload()
		await expect(page.getByLabel('Кратко', { exact: true })).toHaveValue(text)
	} finally {
		await page.goto('/content/rooms')
		await open()
		await page.getByLabel('Кратко', { exact: true }).fill(original)
		await save()
	}
})

test('shows the VIP gallery and adds and removes a photo of the single room', async ({ page }) => {
	await signIn(page)
	const open = async (name: string) => {
		await page.goto('/content/rooms')
		await page
			.getByRole('listitem')
			.filter({ has: page.getByRole('heading', { name, exact: true }) })
			.getByRole('link', { name: /Изменить/ })
			.click()
		await expect(page.getByRole('heading', { name: 'Галерея' })).toBeVisible()
	}
	async function save() {
		const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/rooms/'))
		await page.getByRole('button', { name: 'Сохранить' }).click()
		const res = await saved
		expect(res.ok(), await res.text()).toBe(true)
	}
	const photos = page.getByRole('button', { name: 'Удалить' })

	await open('VIP-номер')
	await expect(photos).toHaveCount(3)

	await open('Одноместный')
	await expect(photos).toHaveCount(0)
	try {
		await page.getByRole('button', { name: 'Добавить фото' }).click()
		await page.getByLabel('Перетащите фото или выберите файл').setInputFiles('e2e/fixtures/photo.jpg')
		await expect(page.getByRole('button', { name: 'Заменить' })).toHaveCount(2)
		await save()
		await page.reload()
		await expect(photos).toHaveCount(1)
		await expect(page.getByRole('button', { name: 'Выше' })).toBeDisabled()
		await expect(page.getByRole('button', { name: 'Ниже' })).toBeDisabled()
	} finally {
		await page.reload()
		while ((await photos.count()) > 0) await photos.first().click()
		await save()
		await page.reload()
		await expect(photos).toHaveCount(0)
	}
})
