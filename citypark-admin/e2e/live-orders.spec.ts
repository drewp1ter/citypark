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

	const cashName = `Тест ${Date.now()}`
	const cash = await request.post(`${apiUrl()}/orders`, {
		data: { phone: newPhone(), name: cashName, paymentType: 'cash', items: [{ productId: product!.id, amount: 1 }] },
	})
	test.skip(cash.status() === 503, 'the restaurant is closed now, orders are not accepted')
	expect(cash.status()).toBe(201)
	const cashId = ((await cash.json()) as { orderId: number }).orderId

	const fresh = page.getByRole('region', { name: 'Новые' })
	await expect(fresh.getByRole('listitem').filter({ hasText: cashName })).toBeVisible()
	// Only a paid order alerts.
	await expect(page.getByRole('region', { name: 'Ждут оплаты' }).getByText(cashName)).toBeHidden()
	await expect(page.getByText('Оплачен заказ')).toBeHidden()

	const onlineName = `Тест онлайн ${Date.now()}`
	const online = await request.post(`${apiUrl()}/orders`, {
		data: {
			phone: newPhone(),
			name: onlineName,
			paymentType: 'online',
			items: [{ productId: product!.id, amount: 1 }],
		},
	})
	expect(online.status()).toBe(201)
	await expect(
		page.getByRole('region', { name: 'Ждут оплаты' }).getByRole('listitem').filter({ hasText: onlineName })
	).toBeVisible()

	await fresh.getByRole('link', { name: `Заказ ${cashId}` }).click()
	await expect(page.getByRole('dialog')).toContainText(`Заказ № ${cashId}`)
	await expect(page.getByRole('dialog')).toContainText(cashName)
})
