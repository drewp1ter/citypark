import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

test('rich text editor follows its profiles', async ({ page }) => {
	await signIn(page)
	await page.goto('/dev/rich-text')
	await expect(page.getByRole('heading', { name: 'Демо: редактор текста' })).toBeVisible()

	const short = page.getByRole('textbox', { name: 'Короткий текст' })
	const article = page.getByRole('textbox', { name: 'Текст статьи' })
	const shortHtml = page.getByTestId('short-html')
	const articleHtml = page.getByTestId('article-html')

	// Bold.
	await short.click()
	await page.keyboard.type('Привет мир')
	await page.keyboard.press('Shift+Control+ArrowLeft')
	await page.getByRole('button', { name: 'Жирный' }).first().click()
	await expect(shortHtml).toContainText('<strong>мир</strong>')

	// The short profile has no heading buttons, the article one makes an h2.
	await expect(page.getByRole('button', { name: 'Заголовок 2' })).toHaveCount(1)
	await article.click()
	await page.keyboard.type('Заголовок')
	await page.getByRole('button', { name: 'Заголовок 2' }).click()
	await expect(articleHtml).toContainText('<h2>Заголовок</h2>')

	// Link.
	await short.click()
	await page.keyboard.press('Control+A')
	await page.getByRole('button', { name: 'Ссылка' }).first().click()
	await page.getByLabel('Адрес ссылки').fill('https://vk.com/cityparkvip')
	await page.getByRole('button', { name: 'Применить' }).click()
	await expect(shortHtml).toContainText('<a href="https://vk.com/cityparkvip">')
	await expect(shortHtml).not.toContainText('target=')

	// Pasting keeps only the allowed formatting.
	await short.click()
	await page.keyboard.press('Control+A')
	await page.keyboard.press('Delete')
	await short.evaluate((el) => {
		const data = new DataTransfer()
		data.setData('text/html', '<h1 style="color:red">Крупно</h1><p style="margin:0">текст</p>')
		el.dispatchEvent(new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }))
	})
	await expect(shortHtml).toContainText('Крупно')
	await expect(shortHtml).not.toContainText('<h1')
	await expect(shortHtml).not.toContainText('style')

	// Price placeholder.
	await short.click()
	await page.keyboard.press('Control+A')
	await page.keyboard.press('Delete')
	await page.getByRole('button', { name: 'Вставить цену' }).click()
	await page.getByRole('button', { name: 'Штраф за курение' }).click()
	await expect(shortHtml).toContainText('{{smokingFine}}')
})
