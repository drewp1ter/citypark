import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import type { components } from '../src/shared/api/schema'
import { signIn } from './helpers'

function apiUrl(): string {
	if (process.env.VITE_API_URL) return process.env.VITE_API_URL
	const env = readFileSync('.env', 'utf8')
	return /^VITE_API_URL=(.+)$/m.exec(env)?.[1]?.trim() ?? 'http://localhost:3000/api/v1'
}

type Product = { id: number; isAvailableForOrder: boolean }

test('new orders are above pending payments on mobile and on the right on desktop', async ({ page }) => {
	await signIn(page)
	await page.getByRole('link', { name: 'Заказы онлайн' }).click()
	const fresh = page.getByRole('region', { name: 'Новые', exact: true })
	const pending = page.getByRole('region', { name: 'Ждут оплаты', exact: true })
	await expect(fresh).toBeVisible()
	await expect(pending).toBeVisible()

	await page.setViewportSize({ width: 390, height: 844 })
	await expect(async () => {
		const freshBox = await fresh.boundingBox()
		const pendingBox = await pending.boundingBox()
		expect(freshBox).not.toBeNull()
		expect(pendingBox).not.toBeNull()
		expect(freshBox!.y + freshBox!.height).toBeLessThanOrEqual(pendingBox!.y)
	}).toPass()

	await page.setViewportSize({ width: 1440, height: 900 })
	await expect(async () => {
		const freshBox = await fresh.boundingBox()
		const pendingBox = await pending.boundingBox()
		expect(freshBox).not.toBeNull()
		expect(pendingBox).not.toBeNull()
		expect(pendingBox!.x + pendingBox!.width).toBeLessThanOrEqual(freshBox!.x)
	}).toPass()
})

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
	await expect(page.getByRole('region', { name: 'Ждут оплаты' })).toBeVisible()
	await expect(page.getByRole('region', { name: 'Новые' })).toBeVisible()
	await expect(page.getByRole('region', { name: 'Подтверждены' })).toHaveCount(0)
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
	await expect(waiting.getByRole('listitem').filter({ hasText: name })).toContainText('Ожидает оплаты')
	await expect(page.getByRole('region', { name: 'Новые' }).getByText(name)).toBeHidden()
	await expect(page.getByText('Оплачен заказ')).toBeHidden()

	const cardTrigger = waiting.getByRole('button', { name: `Заказ ${id}` })
	await cardTrigger.click()
	await expect(page).toHaveURL(/\/orders\/live$/)
	await expect(page.getByRole('dialog')).toContainText(`Заказ № ${id}`)
	await expect(page.getByRole('dialog')).toContainText(name)
	await expect(page.getByRole('button', { name: 'Отправить в IIKO' })).toHaveCount(0)
	await page.getByRole('button', { name: 'Назад к заказам онлайн' }).click()
	await expect(cardTrigger).toBeFocused()
})

test('approves a live order with per-dish comments using a simulated API', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 })
	await page.addInitScript(() => {
		sessionStorage.setItem(
			'citypark-admin-token',
			JSON.stringify({ token: 'test-token', email: 'qa@example.com', name: 'QA' })
		)
	})
	const now = new Date().toISOString()
	let order: components['schemas']['Order'] = {
		id: 42,
		createdAt: now,
		updatedAt: now,
		status: 'paid',
		iikoStatus: 'awaiting_approval',
		comment: '',
		deliveryTime: '',
		total: 500,
		subtotal: 500,
		discount: 0,
		promoCode: null,
		deliveryPrice: 0,
		customer: { id: 1, name: 'Иван', phone: '+79990000000' },
		address: null,
		items: [
			{ productId: 12, name: 'Суп', categoryName: 'Обеды', offeredPrice: 200, amount: 1, discount: 0 },
			{ productId: 13, name: 'Салат', categoryName: 'Обеды', offeredPrice: 300, amount: 1, discount: 0 },
		],
	}
	let attempts = 0
	await page.routeWebSocket('**/orders/live', (ws) => ws.send(JSON.stringify({ type: 'hello' })))
	await page.route(/\/api\/v1\/orders(?:\/42(?:\/approve)?)?(?:\?.*)?$/, async (route) => {
		const pathname = new URL(route.request().url()).pathname
		if (pathname.endsWith('/approve')) {
			expect(route.request().postDataJSON()).toEqual({
				excludedProductIds: [],
				itemComments: [{ productId: 12, comment: 'Без лука' }],
			})
			if (++attempts === 1) {
				await route.fulfill({
					status: 400,
					json: { message: 'Проверьте комментарий', errors: { 'itemComments.12': 'Ошибка комментария' } },
				})
				return
			}
			order = {
				...order,
				iikoStatus: 'queued',
				items: order.items.map((item) => (item.productId === 12 ? { ...item, iikoComment: 'Без лука' } : item)),
			}
		}
		await route.fulfill({ json: pathname.endsWith('/orders') ? { orders: [order], count: 1 } : order })
	})

	await page.goto('/orders/live')
	const trigger = page.getByRole('button', { name: 'Заказ 42' })
	const card = page.getByRole('listitem').filter({ has: trigger })
	await expect(card.getByText('Ожидает подтверждения')).toBeVisible()
	await expect(card.getByRole('link')).toHaveAttribute('href', 'tel:+79990000000')
	await trigger.click()
	const dialog = page.getByRole('dialog')
	const comment = dialog.getByRole('textbox', { name: 'Комментарий для кухни — Суп' })
	const approve = dialog.getByRole('button', { name: 'Отправить в IIKO' })
	await expect(approve).toBeEnabled()
	await comment.fill('я'.repeat(256))
	await expect(approve).toBeDisabled()
	await comment.fill('  Без лука  ')
	await approve.click()
	await expect(dialog.getByText('Ошибка комментария')).toBeVisible()
	await expect(comment).toHaveValue('  Без лука  ')
	await approve.click()
	await expect(dialog.getByText('В очереди на отправку')).toBeVisible()
	await expect(dialog.getByText('Без лука')).toBeVisible()
	await expect(approve).toHaveCount(0)
	await expect(page).toHaveURL(/\/orders\/live$/)
	await dialog.getByRole('button', { name: 'Назад к заказам онлайн' }).click()
	await expect(card.getByText('В очереди на отправку')).toBeVisible()
	order = { ...order, iikoStatus: 'created' }
	await expect(card.getByText('Создан в iiko')).toBeVisible()
	await expect(trigger).toBeFocused()
})
