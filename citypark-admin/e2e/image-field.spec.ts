import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

const PHOTO = 'e2e/fixtures/photo.jpg'
const NOT_AN_IMAGE = 'e2e/fixtures/not-an-image.txt'

test('uploads a photo, moves the focal point and rejects a non-image', async ({ page }) => {
	await signIn(page)
	await page.goto('/dev/image-field')
	await expect(page.getByRole('heading', { name: 'Демо: поле с фото' })).toBeVisible()

	await page.getByLabel('Перетащите фото или выберите файл').setInputFiles(PHOTO)
	const preview = page.getByRole('group', { name: /фокусная точка/ })
	await expect(preview).toBeVisible()
	await expect(page.getByTestId('value')).toContainText('/uploads/')

	// The top-left area gives a point near 10% 10%.
	const box = await preview.boundingBox()
	if (box === null) throw new Error('no preview box')
	await preview.click({ position: { x: box.width * 0.1, y: box.height * 0.1 } })
	const shown = JSON.parse((await page.getByTestId('value').textContent()) ?? '{}') as { position: string }
	const [x = NaN, y = NaN] = shown.position.split(' ').map((p) => parseInt(p, 10))
	expect(Math.abs(x - 10)).toBeLessThanOrEqual(3)
	expect(Math.abs(y - 10)).toBeLessThanOrEqual(3)

	// Arrow keys move the point by 5 %.
	await preview.focus()
	await page.keyboard.press('ArrowRight')
	await expect(page.getByTestId('value')).toContainText(`${x + 5}% ${y}%`)

	await page.getByRole('button', { name: 'Сбросить точку' }).click()
	await expect(page.getByTestId('value')).toContainText('"position":""')

	await page.getByLabel('Описание фото (alt)').fill('Закат над морем')
	await expect(page.getByTestId('value')).toContainText('Закат над морем')

	await page.getByRole('button', { name: 'Убрать фото' }).click()
	await expect(page.getByLabel('Перетащите фото или выберите файл')).toBeAttached()

	await page.getByLabel('Перетащите фото или выберите файл').setInputFiles(NOT_AN_IMAGE)
	await expect(page.getByRole('alert')).toHaveText('Допустимы только JPEG, PNG и WebP')
})
