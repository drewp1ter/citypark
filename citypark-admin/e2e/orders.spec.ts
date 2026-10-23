import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

function apiUrl(): string {
	if (process.env.VITE_API_URL) return process.env.VITE_API_URL
	const env = readFileSync('.env', 'utf8')
	return /^VITE_API_URL=(.+)$/m.exec(env)?.[1]?.trim() ?? 'http://localhost:3000/api/v1'
}

type Product = { id: number; name: string; price: number; isAvailableForOrder: boolean }

const customer = `Тест ${Date.now()}`
// A new phone per run (the backend renames a customer found by phone). Only the digits 7, 8 and 9, so a search by
// the order number rarely matches a phone.
const phone = '+79' + Array.from({ length: 9 }, () => 7 + Math.floor(Math.random() * 3)).join('')
let orderId = 0
let product: Product

test.describe.configure({ mode: 'serial' })

test.beforeAll(async ({ request }) => {
	const products = (await (await request.get(`${apiUrl()}/products`)).json()) as Product[]
	const found = products.find((p) => p.isAvailableForOrder)
	test.skip(found === undefined, 'no product available for order in the dev database')
	product = found!
	const response = await request.post(`${apiUrl()}/orders`, {
		data: { phone, name: customer, paymentType: 'cash', items: [{ productId: product.id, amount: 2 }] },
	})
	test.skip(response.status() === 503, 'the restaurant is closed now, orders are not accepted')
	expect(response.status()).toBe(201)
	orderId = ((await response.json()) as { orderId: number }).orderId
})

test('search, filter, open and go back', async ({ page }) => {
	await signIn(page)
	await page.goto('/orders')
	const search = page.getByLabel('Номер, телефон или имя')

	await search.fill(String(orderId))
	const row = page.getByRole('row').filter({ hasText: customer })
	await expect(row).toBeVisible()
	await expect(page.getByText('Найдено: 1')).toBeVisible()
	await expect(row).toContainText('Самовывоз')
	await expect(row).toContainText('Наличными')
	await expect(row).toContainText('Новый')
	await expect(row.getByRole('link', { name: phone })).toHaveAttribute('href', `tel:${phone}`)

	await search.fill(`нет-такого-${Date.now()}`)
	await expect(page.getByText('Заказов не найдено')).toBeVisible()
	await expect(page.getByRole('button', { name: 'Сбросить фильтры' })).toBeVisible()

	await page.getByRole('button', { name: 'Сбросить фильтры' }).click()
	await expect(search).toHaveValue('')
	await search.fill(customer)
	await page.getByRole('button', { name: 'Сегодня' }).click()
	await expect(row).toBeVisible()
	await expect(page).toHaveURL(/from=\d{4}-\d{2}-\d{2}/)

	await page.getByRole('link', { name: `Заказ ${orderId}` }).click()
	const drawer = page.getByRole('dialog')
	await expect(drawer).toContainText(`Заказ № ${orderId}`)
	await expect(drawer).toContainText(product.name)
	await expect(drawer).toContainText(customer)
	await expect(drawer.getByText('Итого')).toBeVisible()

	await drawer.getByRole('button', { name: 'Назад к списку' }).click()
	await expect(drawer).toBeHidden()
	await expect(page).toHaveURL(/\/orders\?/)
	await expect(page).toHaveURL(/from=/)
	await expect(search).toHaveValue(customer)
	await expect(row).toBeVisible()
})

test('a link reproduces the view, a missing order says so', async ({ page }) => {
	await signIn(page)
	await page.goto(`/orders?status=new&paymentType=cash&q=${encodeURIComponent(customer)}`)
	await expect(page.getByRole('row').filter({ hasText: customer })).toBeVisible()
	await expect(page.getByRole('button', { name: 'Сбросить', exact: true })).toBeVisible()

	await page.goto('/orders/99999999')
	await expect(page.getByRole('dialog').getByText('Заказ не найден')).toBeVisible()
})

test('on a phone the cards open the order', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 800 })
	await signIn(page)
	await page.goto(`/orders?q=${encodeURIComponent(customer)}`)
	const card = page.getByRole('listitem').filter({ hasText: customer })
	await expect(card).toContainText(`№ ${orderId}`)
	await card.getByRole('link', { name: `Заказ ${orderId}` }).click()
	await expect(page.getByRole('dialog')).toContainText(product.name)
})
