import { serverEnv } from '@/shared/config'

export type CreateOrderBody = {
	phone: string
	name: string
	comment: string
	/** "HH:MM", today. Left out for «как можно скорее». */
	deliveryTime?: string
	paymentType: 'online'
	address?: { city: string; address: string }
	items: { productId: number; amount: number }[]
}

export type CreateOrderResult =
	| { ok: true; orderId: number; redirectUrl: string }
	/** `message` is the backend's text for the customer (400 and 503). */
	| { ok: false; message: string }

/** POST /orders. Never cached. A network error or an unexpected status throws. */
export async function createOrder(body: CreateOrderBody): Promise<CreateOrderResult> {
	const response = await fetch(serverEnv().MENU_API_URL + '/orders', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body),
		cache: 'no-store',
	})
	if (response.status === 201) {
		const { orderId, redirectUrl } = (await response.json()) as { orderId: number; redirectUrl: string }
		return { ok: true, orderId, redirectUrl }
	}
	if (response.status === 400 || response.status === 503) {
		const { message } = (await response.json().catch(() => ({}))) as { message?: string }
		if (message) return { ok: false, message }
	}
	throw new Error(`POST /orders: ${response.status}`)
}
