import { readFileSync } from 'node:fs'
import { expect, test, type APIResponse, type Page } from '@playwright/test'
import type { components } from '../src/shared/api/schema'
import { formatRub } from '../src/shared/lib/format'
import { ADMIN_EMAIL, ADMIN_PASSWORD, signIn } from './helpers'

type Order = components['schemas']['Order']
type Campaign = components['schemas']['PromoCampaign']
type CampaignInput = components['schemas']['PromoCampaignInput']
const base = process.env.VITE_API_URL ?? /^VITE_API_URL=(.+)$/m.exec(readFileSync('.env', 'utf8'))?.[1]?.trim()
if (!base) throw new Error('VITE_API_URL is required')
let headers: Record<string, string>
let campaign: Campaign | undefined
const fixtures: Order[] = []
let legacy: Order

async function result<T>(response: APIResponse, status = 200): Promise<T> {
	expect(response.status(), await response.text()).toBe(status)
	return response.json() as Promise<T>
}

test.beforeAll(async ({ request }) => {
	const login = await result<{ token: string }>(
		await request.post(`${base}/auth/sign-in`, { data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD } })
	)
	headers = { Authorization: `Bearer ${login.token}` }
	const products = await result<components['schemas']['Product'][]>(await request.get(`${base}/products?pageSize=500`))
	const available = products.filter((p) => p.isAvailableForOrder && p.price > 10).slice(0, 2)
	expect(available.length, 'Two available dishes are needed for line discount checks').toBe(2)
	const prices = await result<components['schemas']['DeliveryPrice'][]>(await request.get(`${base}/delivery-prices`))
	const city = prices.find((p) => p.price > 0 && p.freeFrom === 0)?.name
	expect(city, 'A town with paid delivery is required').toBeTruthy()
	const input: CampaignInput = {
		code: `QORDER_${Date.now()}`,
		percentage: 50,
		enabled: true,
		target: 'selected',
		productIds: [available[0]!.id],
		categoryIds: [],
		maximumDiscount: 1,
		minimumEligibleSubtotal: null,
		startsAt: null,
		endsAt: null,
	}
	campaign = await result<Campaign>(await request.post(`${base}/promo-codes`, { headers, data: input }), 201)
	try {
		for (const fulfillment of ['pickup', 'delivery'] as const) {
			const items = available.map((p) => ({ productId: p.id, amount: 2 }))
			const quote = await result<components['schemas']['OrderQuote']>(
				await request.post(`${base}/orders/quote`, {
					data: { items, fulfillment, city: fulfillment === 'delivery' ? city : undefined, promoCode: campaign.code },
				})
			)
			expect(quote.discount).toBe(1)
			const created = await result<components['schemas']['CreateOrderResult']>(
				await request.post(`${base}/orders`, {
					data: {
						items,
						name: `Тест промо ${Date.now()}`,
						phone: `+7999${String(Date.now()).slice(-7)}`,
						promoCode: campaign.code,
						expectedPayableTotal: quote.payableTotal,
						address: fulfillment === 'delivery' ? { city, address: 'Тестовый адрес, не доставлять' } : undefined,
						comment: 'QA: неоплаченный тест, не готовить',
					},
				}),
				201
			)
			const order = await result<Order>(await request.get(`${base}/orders/${created.orderId}`, { headers }))
			expect(order.total).toBe(quote.total)
			expect(order.total + order.deliveryPrice).toBe(quote.payableTotal)
			expect(order.items.map((item) => item.discount).sort()).toEqual([0, 1])
			fixtures.push(order)
		}
		const list = await result<components['schemas']['OrderList']>(await request.get(`${base}/orders`, { headers }))
		const noCode = list.orders.find((order) => order.promoCode === null)
		expect(noCode, 'An existing no-code order is required').toBeDefined()
		legacy = noCode!
	} finally {
		// Changing campaign settings must never change the saved order snapshots.
		campaign = await result<Campaign>(
			await request.put(`${base}/promo-codes/${campaign.id}`, {
				headers,
				data: { ...input, percentage: 99, maximumDiscount: 100, enabled: false },
			})
		)
	}
})

test.afterAll(async ({ request }) => {
	if (campaign)
		await result<Campaign>(
			await request.put(`${base}/promo-codes/${campaign.id}`, { headers, data: { ...campaign, enabled: false } })
		)
})

async function expectAmount(page: Page, label: string, value: number) {
	const totals = page.getByRole('dialog').locator('dl[aria-label="Итоги заказа"]')
	await expect(
		totals
			.locator('div')
			.filter({ has: page.locator('dt').filter({ hasText: new RegExp(`^${label}: $`) }) })
			.locator('dd')
	).toHaveText(formatRub(value))
}

for (const width of [390, 1440])
	for (const theme of ['light', 'dark']) {
		test(`stored promo totals ${width}px ${theme}`, async ({ page, request }) => {
			await page.setViewportSize({ width, height: 900 })
			await page.addInitScript((value) => localStorage.setItem('citypark-admin-theme', value), theme)
			await signIn(page)
			await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
			for (const order of fixtures) {
				const snapshot = await result<Order>(await request.get(`${base}/orders/${order.id}`, { headers }))
				expect(snapshot).toEqual(order)
				await page.goto(`/orders/${order.id}`)
				const dialog = page.getByRole('dialog')
				await expect(dialog.getByText(order.promoCode!, { exact: true })).toBeVisible()
				await expectAmount(page, 'Блюда до скидки', order.subtotal)
				await expectAmount(page, 'Скидка по промокоду', -order.discount)
				await expectAmount(page, 'Блюда со скидкой', order.total)
				await expectAmount(page, 'Доставка', order.deliveryPrice)
				await expectAmount(page, 'Итого', order.total + order.deliveryPrice)
				await expect(dialog).toContainText(order.address ? order.address.city : 'Самовывоз')
				for (const item of order.items) {
					const row = dialog.getByRole('row').filter({ hasText: item.name })
					await expect(row.getByRole('gridcell').nth(2)).toHaveText(formatRub(item.offeredPrice))
					await expect(row.getByRole('gridcell').nth(3)).toHaveText(formatRub(item.offeredPrice * item.amount))
					await expect(row.getByRole('gridcell').nth(4)).toHaveText(formatRub(0 - item.discount))
				}
				expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
				await dialog.getByRole('button', { name: 'Назад к списку' }).click()
				await page.goto('/orders/live')
				const card = page
					.getByRole('listitem')
					.filter({ has: page.getByRole('link', { name: `Заказ ${order.id}`, exact: true }) })
				await expect(card).toContainText(formatRub(order.total + order.deliveryPrice))
			}
			await page.goto(`/orders/${legacy.id}`)
			await expect(page.getByRole('dialog').getByRole('columnheader', { name: 'Сумма', exact: true })).toBeVisible()
			await expect(page.getByRole('dialog').getByText('Скидка по промокоду:', { exact: true })).toHaveCount(0)
			await expectAmount(page, 'Доставка', legacy.deliveryPrice)
			await expectAmount(page, 'Итого', legacy.total + legacy.deliveryPrice)
		})
	}
