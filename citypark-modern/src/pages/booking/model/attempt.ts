import { bookingQuoteSchema, type BookingQuote } from '@/entities/room/index.client'
import { createSchema, type BookingAttempt, type BookingValues } from './schema'

export type SavedAttempt = { request: BookingAttempt; values: BookingValues; quote: BookingQuote }
const storageKey = (key: string) => 'citypark-hotel-attempt:' + key
export function rememberAttempt(attempt: SavedAttempt) {
	// Persist before dispatch. Failure leaves the form editable without sending a request.
	sessionStorage.setItem(
		storageKey(attempt.request.idempotencyKey),
		JSON.stringify({ request: attempt.request, quote: attempt.quote })
	)
	const url = new URL(location.href)
	url.searchParams.set('attempt', attempt.request.idempotencyKey)
	history.replaceState(null, '', url)
}
export function restoreAttempt(key: string): SavedAttempt | undefined {
	try {
		const saved: unknown = JSON.parse(sessionStorage.getItem(storageKey(key)) ?? 'null')
		if (!saved || typeof saved !== 'object' || !('request' in saved) || !('quote' in saved)) return undefined
		const request = createSchema.safeParse(saved.request)
		const quote = bookingQuoteSchema.safeParse(saved.quote)
		if (
			!request.success ||
			!quote.success ||
			request.data.idempotencyKey !== key ||
			request.data.quoteId !== quote.data.quoteId
		)
			return undefined
		const { roomTypeId, checkIn, checkOut, guests } = quote.data
		return {
			request: request.data,
			quote: quote.data,
			values: { roomTypeId, checkIn, checkOut, guests, guest: request.data.guest },
		}
	} catch {
		return undefined
	}
}
