import { expect, test, type Page } from '@playwright/test'
import { signIn } from './helpers'

let original: { start: string; end: string } | null = null

async function open(page: Page) {
	await page.goto('/delivery')
	await expect(page.getByRole('heading', { name: 'Часы приёма заказов' })).toBeVisible()
}

async function saveHours(page: Page, start: string, end: string) {
	await page.getByLabel('С', { exact: true }).fill(start)
	await page.getByLabel('До', { exact: true }).fill(end)
	const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/working-hours'))
	await page.getByRole('button', { name: 'Сохранить' }).click()
	const res = await saved
	expect(res.ok(), await res.text()).toBe(true)
}

test.afterEach(async ({ page }) => {
	if (original === null) return
	const { start, end } = original
	original = null
	await open(page)
	if (
		(await page.getByLabel('С', { exact: true }).inputValue()) === start &&
		(await page.getByLabel('До', { exact: true }).inputValue()) === end
	)
		return
	await saveHours(page, start, end)
})

test('changes the working hours, keeps them after a reload and rejects an end before the start', async ({ page }) => {
	await signIn(page)
	await open(page)
	original = {
		start: await page.getByLabel('С', { exact: true }).inputValue(),
		end: await page.getByLabel('До', { exact: true }).inputValue(),
	}

	await saveHours(page, '10:00', '23:00')
	await expect(page.getByText('Сохранено')).toBeVisible()

	await page.reload()
	await expect(page.getByLabel('С', { exact: true })).toHaveValue('10:00')
	await expect(page.getByLabel('До', { exact: true })).toHaveValue('23:00')

	// An end before the start: the error is under the field and nothing is sent.
	let puts = 0
	page.on('request', (r) => {
		if (r.method() === 'PUT' && r.url().includes('/working-hours')) puts++
	})
	await page.getByLabel('До', { exact: true }).fill('09:00')
	await page.getByRole('button', { name: 'Сохранить' }).click()
	await expect(page.getByText('Время окончания должно быть позже начала')).toBeVisible()
	expect(puts).toBe(0)
})
