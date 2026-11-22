'use server'

import { bookingStaySchema, bookingQuoteSchema, bookingUuid } from '@/entities/room'
import { BookingApiError, bookingMessages, bookingRequest, fetchBookingQuote } from '@/entities/room/index.server'
import { createSchema, type BookingState, type BookingValues } from '../model/schema'
import { getBookingContext } from './context'

function failure(error: unknown, values?: BookingValues, key?: string): BookingState {
	if (error instanceof BookingApiError)
		return { status: 'error', message: error.message, code: error.code, values, idempotencyKey: key }
	return { status: 'error', message: bookingMessages.BOOKING_UNAVAILABLE, values, idempotencyKey: key }
}
export async function getQuoteAction(input: unknown): Promise<BookingState> {
	const parsed = bookingStaySchema.safeParse(input)
	if (!parsed.success)
		return {
			status: 'error',
			message: bookingMessages.INVALID_REQUEST,
			errors: Object.fromEntries(parsed.error.issues.map((i) => [i.path[0], i.message])),
		}
	try {
		const context = await getBookingContext(true)
		const quote = await fetchBookingQuote(context!, parsed.data)
		return { status: 'quoted', quote }
	} catch (error) {
		return failure(error)
	}
}
export async function createReservationAction(input: unknown, consent: unknown): Promise<BookingState> {
	const parsed = createSchema.safeParse(input)
	if (!parsed.success)
		return {
			status: 'error',
			message: bookingMessages.INVALID_REQUEST,
			errors: Object.fromEntries(parsed.error.issues.map((i) => [i.path.at(-1), i.message])),
		}
	// Quote and values are display data only: Go verifies the stored quote/context and accepted price.
	const quote = bookingQuoteSchema.safeParse(consent)
	if (!quote.success || quote.data.quoteId !== parsed.data.quoteId)
		return { status: 'error', message: bookingMessages.INVALID_REQUEST }
	const { roomTypeId, checkIn, checkOut, guests } = quote.data
	const values: BookingValues = { roomTypeId, checkIn, checkOut, guests, guest: parsed.data.guest }
	const key = parsed.data.idempotencyKey
	try {
		const context = await getBookingContext()
		if (!context)
			return {
				status: 'unknown',
				message: 'Не удалось восстановить сеанс. Сохраните эту страницу и свяжитесь с отелем.',
				idempotencyKey: key,
				values,
			}
		const result = await bookingRequest('/reservations', context, parsed.data)
		if ('status' in result && result.status === 'confirmed')
			return { status: 'confirmed', reservation: result, idempotencyKey: key }
		return { status: 'pending', idempotencyKey: key, values, quote: quote.data }
	} catch (error) {
		if (error instanceof BookingApiError) return { ...failure(error, values, key), quote: quote.data }
		// A lost response can follow a committed booking. Retain the exact attempt for recovery.
		return {
			status: 'pending',
			message: 'Ответ задерживается. Проверяем бронирование.',
			idempotencyKey: key,
			values,
			quote: quote.data,
		}
	}
}
export async function lookupReservationAction(input: unknown): Promise<BookingState> {
	const parsed = bookingUuid.safeParse(input)
	if (!parsed.success) return { status: 'error', message: bookingMessages.INVALID_REQUEST }
	const key = parsed.data
	try {
		const context = await getBookingContext()
		if (!context)
			return {
				status: 'unknown',
				idempotencyKey: key,
				message: 'Не удалось восстановить сеанс. Свяжитесь с отелем перед повторным бронированием.',
			}
		const result = await bookingRequest('/reservations/' + key, context)
		if ('status' in result && result.status === 'confirmed')
			return { status: 'confirmed', reservation: result, idempotencyKey: key }
		return { status: 'pending', idempotencyKey: key }
	} catch (error) {
		if (
			error instanceof BookingApiError &&
			error.code !== 'RESERVATION_NOT_FOUND' &&
			error.code !== 'BOOKING_UNAVAILABLE' &&
			error.code !== 'RATE_LIMITED'
		)
			return failure(error, undefined, key)
		return {
			status: 'unknown',
			idempotencyKey: key,
			message:
				error instanceof BookingApiError && error.code === 'RESERVATION_NOT_FOUND'
					? 'Результат пока не найден. Повторите тот же запрос или свяжитесь с отелем.'
					: 'Не удалось проверить бронирование. Сохраняем запрос и продолжим проверку.',
		}
	}
}
