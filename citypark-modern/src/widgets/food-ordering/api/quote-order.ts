'use server'

import { z } from 'zod'
import { getMenu, priceLines } from '@/entities/dish'
import { linesSchema, methods } from '../model/schema'
import { fetchOrderQuote } from './orders-api'

const inputSchema = z.object({
	lines: linesSchema,
	method: z.enum(methods),
	city: z.string().trim().max(80),
	promoCode: z.string().trim().max(256),
})

/** Resolve only real menu IDs on the server; the backend owns all pricing and promo rules. */
export async function quoteOrder(input: unknown) {
	const parsed = inputSchema.safeParse(input)
	if (!parsed.success) return { ok: false as const, message: 'Проверьте корзину и данные доставки.' }
	try {
		const { lines, method, city, promoCode } = parsed.data
		const menu = await getMenu()
		const priced = priceLines(lines, menu.dishes)
		if (priced.unavailable.length > 0)
			return {
				ok: false as const,
				unavailableLines: priced.unavailable.map(({ dishId, variantId }) => ({ dishId, variantId })),
			}
		return await fetchOrderQuote({
			items: priced.items.map((item) => ({ productId: Number(item.dish.externalId), amount: item.quantity })),
			fulfillment: method,
			...(method === 'delivery' ? { city } : {}),
			promoCode,
		})
	} catch (error) {
		console.error(error)
		return { ok: false as const, message: 'Не удалось рассчитать заказ. Попробуйте ещё раз.' }
	}
}
