import { expect, test, type Page } from '@playwright/test'
import { signIn } from './helpers'

async function openDelivery(page: Page) {
	await signIn(page)
	await page.goto('/delivery')
	await expect(page.getByRole('heading', { name: 'Доставка' })).toBeVisible()
}

async function addCity(page: Page, name: string, price: string) {
	await page.getByRole('button', { name: 'Добавить город' }).click()
	const dialog = page.getByRole('dialog')
	await dialog.getByLabel('Название города').fill(name)
	await dialog.getByLabel('Цена доставки, ₽').fill(price)
	await dialog.getByRole('button', { name: 'Сохранить' }).click()
}

test('add, duplicate, edit and delete a delivery price', async ({ page }) => {
	const name = `Тест ${Date.now()}`
	await openDelivery(page)

	await addCity(page, name, '123')
	await expect(page.getByText('Сохранено')).toBeVisible()
	const row = page.getByRole('row').filter({ hasText: name })
	await expect(row).toContainText('123 ₽')

	await addCity(page, name, '123')
	await expect(page.getByRole('dialog').getByText('Такой город уже есть')).toBeVisible()
	await page.getByRole('dialog').getByRole('button', { name: 'Отмена' }).click()
	await expect(page.getByRole('dialog')).toBeHidden()

	await page.getByRole('button', { name: `Изменить ${name}` }).click()
	const edit = page.getByRole('dialog')
	await edit.getByLabel('Цена доставки, ₽').fill('150')
	await edit.getByRole('button', { name: 'Сохранить' }).click()
	await expect(row).toContainText('150 ₽')

	await page.getByRole('button', { name: `Удалить ${name}` }).click()
	await page.getByRole('alertdialog').getByRole('button', { name: 'Удалить' }).click()
	await expect(row).toBeHidden()
})

test('client validation keeps what the user typed', async ({ page }) => {
	await openDelivery(page)
	await page.getByRole('button', { name: 'Добавить город' }).click()
	const dialog = page.getByRole('dialog')
	await dialog.getByLabel('Цена доставки, ₽').fill('0')
	await dialog.getByRole('button', { name: 'Сохранить' }).click()
	await expect(dialog.getByText('Укажите название города')).toBeVisible()
	await expect(dialog.getByText('Цена должна быть больше 0')).toBeVisible()
	await expect(dialog.getByLabel('Цена доставки, ₽')).toHaveValue('0')
})

test('on a phone the cards show the same data', async ({ page }) => {
	const name = `Тест ${Date.now()}`
	await openDelivery(page)
	await addCity(page, name, '123')
	await expect(page.getByRole('row').filter({ hasText: name })).toBeVisible()

	await page.setViewportSize({ width: 390, height: 800 })
	const card = page.getByRole('listitem').filter({ hasText: name })
	await expect(card).toContainText('123 ₽')
	await card.getByRole('button', { name: `Удалить ${name}` }).click()
	await page.getByRole('alertdialog').getByRole('button', { name: 'Удалить' }).click()
	await expect(card).toBeHidden()
})
