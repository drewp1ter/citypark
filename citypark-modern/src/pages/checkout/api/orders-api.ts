import { serverEnv } from '@/shared/config'
import { getCustomerSessionToken, getGuestCheckoutCookie, saveGuestCheckoutCookie } from '@/shared/customer'

export type CreateOrderBody = {
	phone: string
	name: string
	comment: string
	/** "HH:MM", today. Left out for «как можно скорее». */
	deliveryTime?: string
	address?: { city: string; address: string }
	items: { productId: number; amount: number }[]
}

export type CreateOrderResult =
	| { ok: true; orderId: number; redirectUrl: string }
	/** `message` is the backend's text for the customer (400 and 503); `fields` are the 400's per-field `errors`. */
	| { ok: false; message: string; fields?: Record<string, string> }

/** POST /orders. Never cached. A network error or an unexpected status throws. */
export async function createOrder(body: CreateOrderBody): Promise<CreateOrderResult> {
	const token = await getCustomerSessionToken()
	const guest = token ? undefined : await getGuestCheckoutCookie()
	const response = await fetch(serverEnv().MENU_API_URL + '/orders', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			...(token ? { Authorization: `Bearer ${token}` } : guest ? { Cookie: `citypark_guest_checkout=${guest}` } : {}),
		},
		body: JSON.stringify(body),
		cache: 'no-store',
	})
	if (!token) await saveGuestCheckoutCookie(response.headers.get('set-cookie'))
	if (response.status === 401) return { ok: false, message: 'Войдите повторно, чтобы продолжить' }
	if (response.status === 201) {
		const { orderId, redirectUrl } = (await response.json()) as { orderId: number; redirectUrl: string }
		return { ok: true, orderId, redirectUrl }
	}
	if (response.status === 400 || response.status === 503) {
		const { message, errors } = (await response.json().catch(() => ({}))) as {
			message?: string
			errors?: Record<string, string>
		}
		if (message) return { ok: false, message, fields: errors }
	}
	throw new Error(`POST /orders: ${response.status}`)
}
