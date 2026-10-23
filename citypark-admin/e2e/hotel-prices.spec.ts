import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

const SAVE = 'Сохранить'

test('changes the smoking fine, keeps it after a reload, then restores it', async ({ page }) => {
	await signIn(page)
	await page.goto('/content/hotel-prices')
	await expect(page.getByRole('heading', { name: 'Цены отеля' })).toBeVisible()
	await expect(page.getByText('Используется в текстах как {{smokingFine}}')).toBeVisible()

	const fine = page.getByLabel('Штраф за курение, ₽')
	const save = page.getByRole('button', { name: SAVE })
	await expect(save).toBeDisabled()
	const original = await fine.inputValue()
	const changed = original === '2345,5' ? '2346' : '2345,5'

	try {
		await fine.fill(changed)
		await save.click()
		await expect(page.getByText('Сохранено')).toBeVisible()
		await page.reload()
		await expect(page.getByLabel('Штраф за курение, ₽')).toHaveValue(changed)
	} finally {
		await page.goto('/content/hotel-prices')
		await page.getByLabel('Штраф за курение, ₽').fill(original)
		const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/hotel/prices'))
		await page.getByRole('button', { name: SAVE }).click()
		expect((await saved).ok()).toBe(true)
	}
})

test('adds a breakfast item, moves it up, keeps the order, then deletes it', async ({ page }) => {
	await signIn(page)
	await page.goto('/content/hotel-prices')
	await expect(page.getByRole('heading', { name: 'Цены отеля' })).toBeVisible()
	const name = `Тест ${Date.now()}`
	const rows = page.getByRole('group', { name: /^Блюдо \d+$/ })
	const before = await rows.count()

	async function saveAndWait() {
		const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/hotel/prices'))
		await page.getByRole('button', { name: SAVE }).click()
		expect((await saved).ok()).toBe(true)
	}

	await page.getByRole('button', { name: 'Добавить блюдо' }).click()
	const last = rows.nth(before)
	await last.getByLabel('Название', { exact: true }).fill(name)
	await last.getByLabel('Цена, ₽').fill('250,5')
	await last.getByText('Только на компьютере').click()
	await expect(last.getByLabel('Только на компьютере')).toBeChecked()
	if (before > 0) await page.getByRole('button', { name: `Блюдо ${before + 1} выше` }).click()
	const position = before > 0 ? before - 1 : 0
	await expect(rows.nth(position).getByLabel('Название', { exact: true })).toHaveValue(name)

	try {
		await saveAndWait()
		await page.reload()
		await expect(rows.nth(position).getByLabel('Название', { exact: true })).toHaveValue(name)
		await expect(rows.nth(position).getByLabel('Цена, ₽')).toHaveValue('250,5')
		await expect(rows.nth(position).getByLabel('Только на компьютере')).toBeChecked()
		await expect(rows).toHaveCount(before + 1)
	} finally {
		await page.goto('/content/hotel-prices')
		const index = await rows.evaluateAll(
			(els, wanted) => els.findIndex((el) => Array.from(el.querySelectorAll('input')).some((i) => i.value === wanted)),
			name
		)
		if (index >= 0) {
			await page.getByRole('button', { name: `Удалить блюдо ${index + 1}` }).click()
			await saveAndWait()
			await page.reload()
			await expect(rows).toHaveCount(before)
		}
	}
})
