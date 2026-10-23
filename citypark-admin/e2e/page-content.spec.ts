import { expect, test, type Page } from '@playwright/test'
import { signIn } from './helpers'

/** The card of one block, found by its key caption. */
function card(page: Page, key: string) {
	return page.getByRole('article').filter({ has: page.getByText(key, { exact: true }) })
}

test('changes a text block on the home page and keeps it after a reload', async ({ page }) => {
	const key = 'home.rooms.heading'
	await signIn(page)
	await page.goto('/content/home')
	await expect(page.getByRole('heading', { name: 'Главная', exact: true })).toBeVisible()
	await expect(page.getByRole('link', { name: 'Открыть на сайте ↗' })).toBeVisible()

	const block = card(page, key)
	const input = block.getByRole('textbox')
	const original = await input.inputValue()
	const changed = `Тест ${Date.now()}`
	const save = block.getByRole('button', { name: /^Сохранить/ })
	try {
		await expect(save).toBeDisabled()
		await input.fill(changed)
		await expect(save).toBeEnabled()
		await expect(block.getByRole('img', { name: 'Есть несохранённые изменения' })).toBeVisible()
		await save.click()
		await expect(page.getByText('Сохранено')).toBeVisible()
		await expect(block.getByText(/^изменено \d{2}\.\d{2}\.\d{4}, \d{2}:\d{2}$/)).toBeVisible()
		await expect(save).toBeDisabled()

		await page.reload()
		await expect(card(page, key).getByRole('textbox')).toHaveValue(changed)
	} finally {
		await page.goto('/content/home')
		await card(page, key).getByRole('textbox').fill(original)
		const restore = card(page, key).getByRole('button', { name: /^Сохранить/ })
		if (await restore.isEnabled()) await restore.click()
		await expect(card(page, key).getByRole('textbox')).toHaveValue(original)
	}
})

test('adds a line to a list on the hotel page and removes it again', async ({ page }) => {
	const key = 'hotel.amenities'
	const line = `Тест ${Date.now()}`
	await signIn(page)
	await page.goto('/content/hotel')
	await expect(page.getByRole('heading', { name: 'Отель', exact: true })).toBeVisible()

	const block = card(page, key)
	const lines = block.getByRole('listitem')
	const before = await lines.count()
	const save = block.getByRole('button', { name: /^Сохранить/ })

	await block.getByRole('button', { name: 'Добавить строку' }).click()
	await expect(lines).toHaveCount(before + 1)
	await lines.last().getByRole('textbox').fill(line)
	await save.click()
	await expect(page.getByText('Сохранено')).toBeVisible()

	await page.reload()
	await expect(card(page, key).getByRole('listitem')).toHaveCount(before + 1)
	await expect(card(page, key).getByRole('listitem').last().getByRole('textbox')).toHaveValue(line)

	await card(page, key)
		.getByRole('listitem')
		.last()
		.getByRole('button', { name: /^Удалить строку/ })
		.click()
	await expect(card(page, key).getByRole('listitem')).toHaveCount(before)
	await card(page, key)
		.getByRole('button', { name: /^Сохранить/ })
		.click()
	await expect(page.getByText('Сохранено').last()).toBeVisible()
	await page.reload()
	await expect(card(page, key).getByRole('listitem')).toHaveCount(before)
})

test('shows the server error for an unknown price placeholder', async ({ page }) => {
	const key = 'home.rooms.heading'
	await signIn(page)
	await page.goto('/content/home')

	const block = card(page, key)
	await block.getByRole('textbox').fill('Номера {{price}}')
	await block.getByRole('button', { name: /^Сохранить/ }).click()
	await expect(block.getByText(/неизвестная подстановка/)).toBeVisible()
	// Nothing was lost: the typed text is still there and still unsaved.
	await expect(block.getByRole('textbox')).toHaveValue('Номера {{price}}')
	await expect(block.getByRole('button', { name: /^Сохранить/ })).toBeEnabled()
})
