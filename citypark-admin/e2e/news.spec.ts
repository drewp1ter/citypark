import { expect, test, type Page } from '@playwright/test'
import { signIn } from './helpers'

const PHOTO = 'e2e/fixtures/photo.jpg'

function todayRu(): string {
	return new Intl.DateTimeFormat('ru-RU', {
		timeZone: 'Europe/Moscow',
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
	}).format(new Date())
}

function row(page: Page, title: string) {
	return page.getByRole('listitem').filter({ has: page.getByRole('heading', { name: title, exact: true }) })
}

async function removeNews(page: Page, title: string) {
	await page.goto('/content/news')
	const item = row(page, title)
	if ((await item.count()) === 0) return
	await item.getByRole('button', { name: /^Удалить/ }).click()
	await page.getByRole('alertdialog').getByRole('button', { name: 'Удалить' }).click()
	await expect(row(page, title)).toHaveCount(0)
}

test('creates, edits and deletes news, and rejects a taken address', async ({ page }) => {
	const stamp = Date.now()
	const title = `Тест ${stamp}`
	const renamed = `Тест ${stamp} новый`
	const second = `Тест ${stamp} второй`
	const slug = `test-${stamp}`

	await signIn(page)
	try {
		await page.goto('/content/news')
		await expect(page.getByRole('heading', { name: 'Новости', exact: true })).toBeVisible()
		await expect(page.getByText(/^Найдено: \d+$/)).toBeVisible()

		// Create with a bold word and the photo.
		await page.getByRole('link', { name: 'Добавить новость' }).click()
		await page.getByLabel('Заголовок', { exact: true }).fill(title)
		await page.getByLabel('Адрес на сайте').fill(slug)
		await page.getByLabel('Перетащите фото или выберите файл').setInputFiles(PHOTO)
		await expect(page.getByRole('button', { name: 'Убрать фото' })).toBeVisible()
		await page.getByLabel('Описание фото').fill('Тестовое фото')
		const body = page.getByRole('textbox', { name: 'Текст', exact: true })
		await body.click()
		await page.keyboard.type('Тест жирный')
		await page.keyboard.press('Shift+Control+ArrowLeft')
		await page.getByRole('button', { name: 'Жирный' }).click()
		await page.getByRole('button', { name: 'Опубликовать' }).click()
		await expect(page.getByText('Новость опубликована')).toBeVisible()
		await expect(page).toHaveURL(new RegExp(`/content/news/${slug}$`))

		// It is first in the list with today's date.
		await page.getByRole('link', { name: '← Новости' }).click()
		const first = page
			.getByRole('listitem')
			.filter({ has: page.getByRole('heading', { level: 2 }) })
			.first()
		await expect(first.getByRole('heading', { name: title, exact: true })).toBeVisible()
		await expect(first).toContainText(todayRu())

		// Opening it shows the bold word and the photo.
		await first.getByRole('link', { name: /^Изменить/ }).click()
		await page.reload()
		await expect(page.getByLabel('Заголовок', { exact: true })).toHaveValue(title)
		await expect(page.getByRole('textbox', { name: 'Текст', exact: true }).locator('strong')).toHaveCount(1)
		await expect(page.getByRole('button', { name: 'Убрать фото' })).toBeVisible()

		// Changing the title saves.
		await page.getByLabel('Заголовок', { exact: true }).fill(renamed)
		await page.getByRole('button', { name: 'Сохранить' }).click()
		await expect(page.getByText('Сохранено')).toBeVisible()
		await page.getByRole('link', { name: '← Новости' }).click()
		await expect(row(page, renamed)).toBeVisible()

		// A second item with the same explicit address is refused.
		await page.getByRole('link', { name: 'Добавить новость' }).click()
		await page.getByLabel('Заголовок', { exact: true }).fill(second)
		await page.getByLabel('Адрес на сайте').fill(slug)
		await page.getByRole('button', { name: 'Опубликовать' }).click()
		await expect(page.getByText('Такой адрес уже занят')).toBeVisible()

		// With an empty address it is accepted (the server makes a free one).
		await page.getByLabel('Адрес на сайте').fill('')
		await page.getByRole('button', { name: 'Опубликовать' }).click()
		await expect(page.getByText('Новость опубликована')).toBeVisible()
		await page.getByRole('link', { name: '← Новости' }).click()
		await expect(row(page, second)).toBeVisible()
	} finally {
		await removeNews(page, renamed)
		await removeNews(page, title)
		await removeNews(page, second)
	}

	await page.goto('/content/news')
	await expect(row(page, renamed)).toHaveCount(0)
	await expect(row(page, second)).toHaveCount(0)
})
