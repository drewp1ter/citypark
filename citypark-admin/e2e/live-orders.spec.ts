import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

function apiUrl(): string {
	if (process.env.VITE_API_URL) return process.env.VITE_API_URL
	const env = readFileSync('.env', 'utf8')
	return /^VITE_API_URL=(.+)$/m.exec(env)?.[1]?.trim() ?? 'http://localhost:3000/api/v1'
}

type Product = { id: number; isAvailableForOrder: boolean }

// A new phone per run (the backend renames a customer found by phone).
function newPhone(): string {
	return '+79' + Array.from({ length: 9 }, () => 7 + Math.floor(Math.random() * 3)).join('')
}

test('new orders appear without a reload and open their details', async ({ page, request }) => {
	const products = (await (await request.get(`${apiUrl()}/products`)).json()) as Product[]
	const product = products.find((p) => p.isAvailableForOrder)
	test.skip(product === undefined, 'no product available for order in the dev database')

	await signIn(page)
	await page.getByRole('link', { name: 'Заказы онлайн' }).click()
	await expect(page.getByRole('heading', { name: 'Заказы онлайн' })).toBeVisible()
	await expect(page.getByText('Онлайн', { exact: true })).toBeVisible()

	const name = `Тест ${Date.now()}`
	const created = await request.post(`${apiUrl()}/orders`, {
		data: { phone: newPhone(), name, items: [{ productId: product!.id, amount: 1 }] },
	})
	test.skip(created.status() === 503, 'the restaurant is closed now, orders are not accepted')
	expect(created.status()).toBe(201)
	const id = ((await created.json()) as { orderId: number }).orderId

	// An unpaid order waits for the payment: it is not in «Новые» and does not alert.
	const waiting = page.getByRole('region', { name: 'Ждут оплаты' })
	await expect(waiting.getByRole('listitem').filter({ hasText: name })).toBeVisible()
	await expect(page.getByRole('region', { name: 'Новые' }).getByText(name)).toBeHidden()
	await expect(page.getByText('Оплачен заказ')).toBeHidden()

	await waiting.getByRole('link', { name: `Заказ ${id}` }).click()
	await expect(page.getByRole('dialog')).toContainText(`Заказ № ${id}`)
	await expect(page.getByRole('dialog')).toContainText(name)
})
