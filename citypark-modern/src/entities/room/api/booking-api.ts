import 'server-only'
import { randomBytes } from 'node:crypto'
import { z } from 'zod'
import { serverEnv } from '@/shared/config'
import {
	bookingQuoteSchema,
	bookingReservationSchema,
	bookingStaySchema,
	bookingRoomIds,
	bookingCapacities,
	type BookingStay,
} from '../model/booking'
import type { RoomAvailability, StayQuery } from '../model/types'

export const bookingMessages = {
	INVALID_REQUEST: 'Проверьте данные бронирования',
	RESERVATION_NOT_FOUND: 'Бронирование не найдено',
	ROOM_UNAVAILABLE: 'На выбранные даты нет свободного номера',
	PRICE_CHANGED: 'Стоимость проживания изменилась. Подтвердите новую стоимость',
	IDEMPOTENCY_CONFLICT: 'Ключ бронирования уже использован с другими данными',
	QUOTE_EXPIRED: 'Расчёт стоимости устарел. Обновите стоимость',
	RATE_LIMITED: 'Слишком много запросов. Попробуйте позже',
	BOOKING_UNAVAILABLE: 'Бронирование временно недоступно',
} as const
export type BookingErrorCode = keyof typeof bookingMessages
export class BookingApiError extends Error {
	constructor(
		public code: BookingErrorCode,
		public status: number
	) {
		super(bookingMessages[code])
	}
}
const errorSchema = z
	.object({
		statusCode: z.number(),
		code: z.enum(Object.keys(bookingMessages) as [BookingErrorCode, ...BookingErrorCode[]]),
		message: z.string(),
	})
	.strict()
const pendingSchema = z.object({ status: z.literal('pending') }).strict()

/** Never forward browser headers, credentials or response bodies into logs. */
export async function bookingRequest(path: string, context: string, body?: unknown) {
	const response = await fetch(serverEnv().MENU_API_URL + '/hotel' + path, {
		method: body === undefined ? 'GET' : 'POST',
		cache: 'no-store',
		redirect: 'error',
		headers: { 'Content-Type': 'application/json', 'X-Hotel-Booking-Context': context },
		body: body === undefined ? undefined : JSON.stringify(body),
		signal: AbortSignal.timeout(20_000),
	})
	const reader = response.body?.getReader()
	if (!reader) throw new Error('Missing booking response')
	const chunks: Uint8Array[] = []
	let length = 0
	try {
		while (true) {
			const { value, done } = await reader.read()
			if (done) break
			length += value.length
			if (length > 65_536) throw new Error('Oversized booking response')
			chunks.push(value)
		}
	} finally {
		await reader.cancel()
	}
	const data: unknown = JSON.parse(Buffer.concat(chunks).toString('utf8'))
	if (!response.ok) {
		const error = errorSchema.safeParse(data)
		if (!error.success || error.data.statusCode !== response.status) throw new Error('Invalid booking error')
		throw new BookingApiError(error.data.code, response.status)
	}
	if (path === '/quotes' && response.status === 200) return bookingQuoteSchema.parse(data)
	if (path !== '/quotes' && response.status === 202) return pendingSchema.parse(data)
	if (path !== '/quotes' && (response.status === 200 || response.status === 201))
		return bookingReservationSchema.parse(data)
	throw new Error('Unexpected booking status')
}
export async function fetchBookingQuote(context: string, stay: BookingStay) {
	const quote = bookingQuoteSchema.parse(await bookingRequest('/quotes', context, stay))
	if (
		quote.roomTypeId !== stay.roomTypeId ||
		quote.checkIn !== stay.checkIn ||
		quote.checkOut !== stay.checkOut ||
		quote.guests !== stay.guests
	)
		throw new Error('Mismatched booking quote')
	return quote
}

/** Full-stay advisory checks. Each unknown outcome stays unknown, including rate limits/outages. */
export async function fetchStayAvailability(query: StayQuery): Promise<Record<string, RoomAvailability>> {
	const context = randomBytes(32).toString('base64url')
	const entries = await Promise.all(
		bookingRoomIds.map(async (roomTypeId) => {
			if (query.guests > bookingCapacities[roomTypeId]) return [roomTypeId, { state: 'unavailable' }] as const
			const stay = bookingStaySchema.safeParse({ ...query, roomTypeId })
			if (!stay.success) return [roomTypeId, { state: 'unknown' }] as const
			try {
				const quote = await fetchBookingQuote(context, stay.data)
				return [roomTypeId, { state: 'available', price: quote.price }] as const
			} catch (error) {
				return [
					roomTypeId,
					{ state: error instanceof BookingApiError && error.code === 'ROOM_UNAVAILABLE' ? 'unavailable' : 'unknown' },
				] as const
			}
		})
	)
	return Object.fromEntries(entries)
}
