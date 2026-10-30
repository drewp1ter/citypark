'use server'

import { redirect } from 'next/navigation'
import { getDeliveryTowns, getMenu, priceLines } from '@/entities/dish'
import { getWorkingHours } from '@/shared/api'
import { phoneDigits } from '@/shared/lib'
import { setPendingPaymentContext } from '@/shared/customer'
import {
	conditionalErrors,
	linesSchema,
	orderFields,
	orderSchema,
	type OrderErrorKey,
	type OrderField,
	type OrderState,
} from '../model/schema'
import { createOrder, type CreateOrderBody } from './orders-api'

/**
 * Validates the order, checks the cart against the menu (never trusting the browser's prices) and places it with the
 * backend's `POST /orders`. On success it redirects to the Moneta payment page; the cart is cleared on `/checkout/done`.
 */
export async function placeOrder(previous: OrderState, formData: FormData): Promise<OrderState> {
	return submitOrder('web', previous, formData)
}

export async function placeMiniOrder(previous: OrderState, formData: FormData): Promise<OrderState> {
	return submitOrder('mini', previous, formData)
}

async function submitOrder(surface: 'web' | 'mini', _previous: OrderState, formData: FormData): Promise<OrderState> {
	const values = Object.fromEntries(orderFields.map((field) => [field, String(formData.get(field) ?? '')])) as Record<
		OrderField,
		string
	>
	const errors: Partial<Record<OrderErrorKey, string>> = {}

	const order = orderSchema.safeParse(values)
	if (!order.success) {
		for (const issue of order.error.issues) {
			const field = issue.path[0] as OrderField
			errors[field] ??= issue.message
		}
	}
	const hours = await getWorkingHours()
	for (const [field, message] of Object.entries(conditionalErrors(values, hours)) as [OrderField, string][]) {
		errors[field] ??= message
	}

	let rawLines: unknown = null
	try {
		rawLines = JSON.parse(String(formData.get('lines') ?? '[]'))
	} catch {
		// handled below: the schema rejects null
	}
	const lines = linesSchema.safeParse(rawLines)
	if (!lines.success) errors.form = 'Корзина пуста. Добавьте блюда из меню.'

	if (!order.success || !lines.success || Object.keys(errors).length > 0) return { status: 'error', errors, values }

	let menu
	try {
		menu = await getMenu()
	} catch (error) {
		console.error(error)
		errors.form = 'Не удалось проверить меню. Попробуйте ещё раз через минуту.'
		return { status: 'error', errors, values }
	}
	if (lines.data.some((line) => menu.dishes.find((dish) => dish.id === line.dishId)?.orderBlocked)) {
		errors.form = 'Удалите недоступные блюда из корзины, чтобы оформить заказ.'
		return { status: 'error', errors, values }
	}
	const priced = priceLines(lines.data, menu.dishes)
	if (priced.items.length !== lines.data.length) {
		// A dish was removed from the menu, is on the stop-list or has no price since it was put in the cart.
		errors.form = 'Меню изменилось: некоторых блюд из корзины больше нет. Проверьте заказ.'
		return { status: 'error', errors, values }
	}

	if (order.data.method === 'delivery') {
		let towns
		try {
			towns = await getDeliveryTowns()
		} catch (error) {
			console.error(error)
			errors.form = 'Не удалось проверить адрес доставки. Попробуйте ещё раз через минуту.'
			return { status: 'error', errors, values }
		}
		if (!towns.some((town) => town.name === order.data.city)) {
			errors.city = 'Выберите населённый пункт'
			return { status: 'error', errors, values }
		}
	}

	const { street, apartment, entrance, city, when, time, comment } = order.data
	const body: CreateOrderBody = {
		phone: `+${phoneDigits(order.data.phone)}`,
		name: order.data.name,
		comment,
		items: priced.items.map((item) => ({ productId: Number(item.dish.externalId), amount: item.quantity })),
	}
	if (when === 'scheduled') body.deliveryTime = time
	if (order.data.method === 'delivery') {
		const address = [street, apartment && `кв. ${apartment}`, entrance && `подъезд ${entrance}`].filter(Boolean)
		body.address = { city, address: address.join(', ') }
	}

	let redirectUrl: string
	try {
		const result = await createOrder(body, surface)
		if (!result.ok) {
			if (result.fields?.deliveryTime) errors.time = result.fields.deliveryTime
			else errors.form = result.message
			return { status: 'error', errors, values }
		}
		redirectUrl = result.redirectUrl
		await setPendingPaymentContext(result.orderId, surface)
	} catch (error) {
		console.error(error)
		errors.form = 'Не удалось отправить заказ. Попробуйте ещё раз через минуту.'
		return { status: 'error', errors, values }
	}

	// redirect() works by throwing, so it must stay outside any try/catch.
	redirect(redirectUrl)
}
