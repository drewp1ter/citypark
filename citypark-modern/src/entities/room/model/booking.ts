import { z } from 'zod'
import { todayInMoscow } from '@/shared/lib'

export const bookingCapacities = { single: 1, 'single-balcony': 1, 'double-twin': 2, 'double-euro': 2, vip: 4 } as const
export const bookingRoomIds = ['single', 'single-balcony', 'double-twin', 'double-euro', 'vip'] as const
export const bookingUuid = z
	.string()
	.regex(/^(?!00000000-0000-0000-0000-000000000000)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/)
export function isBookingDate(value: string): boolean {
	if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || Number(value.slice(0, 4)) < 1 || Number(value.slice(0, 4)) > 3999)
		return false
	const date = new Date(value + 'T00:00:00Z')
	return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === value
}
export const bookingStaySchema = z
	.object({
		roomTypeId: z.enum(bookingRoomIds, { error: 'Выберите тип номера' }),
		checkIn: z.string({ error: 'Укажите дату заезда' }).refine(isBookingDate, 'Укажите дату заезда'),
		checkOut: z.string({ error: 'Укажите дату выезда' }).refine(isBookingDate, 'Укажите дату выезда'),
		guests: z
			.number({ error: 'Укажите число гостей от 1 до 4' })
			.int('Укажите целое число гостей')
			.min(1, 'Укажите число гостей от 1 до 4')
			.max(4, 'Укажите число гостей от 1 до 4'),
	})
	.strict()
	.superRefine((stay, ctx) => {
		if (stay.checkIn <= todayInMoscow())
			ctx.addIssue({ code: 'custom', path: ['checkIn'], message: 'Заезд возможен с завтрашнего дня' })
		if (stay.checkOut <= stay.checkIn)
			ctx.addIssue({ code: 'custom', path: ['checkOut'], message: 'Выезд должен быть позже заезда' })
		if (stay.guests > bookingCapacities[stay.roomTypeId])
			ctx.addIssue({ code: 'custom', path: ['guests'], message: 'Слишком много гостей для выбранного номера' })
	})
const priceSchema = z
	.object({
		currency: z.literal('RUB'),
		nights: z.number().int().positive(),
		pricePerNight: z.number().positive().finite(),
		total: z.number().positive().finite(),
	})
	.strict()
export const bookingQuoteSchema = z
	.object({
		quoteId: bookingUuid,
		expiresAt: z.iso.datetime({ offset: true }),
		roomTypeId: z.enum(bookingRoomIds),
		checkIn: z.string().refine(isBookingDate),
		checkOut: z.string().refine(isBookingDate),
		guests: z.number().int().min(1).max(4),
		available: z.literal(true),
		price: priceSchema,
	})
	.strict()
export const bookingReservationSchema = z
	.object({
		reservationId: bookingUuid,
		status: z.literal('confirmed'),
		roomTypeId: z.enum(bookingRoomIds),
		checkIn: z.string().refine(isBookingDate),
		checkOut: z.string().refine(isBookingDate),
		guests: z.number().int().min(1).max(4),
		price: priceSchema,
		confirmationEmailStatus: z.enum(['pending', 'sent']),
	})
	.strict()
export type BookingStay = z.infer<typeof bookingStaySchema>
export type BookingQuote = z.infer<typeof bookingQuoteSchema>
export type BookingReservation = z.infer<typeof bookingReservationSchema>
export function bookingHref(stay: BookingStay): string {
	return '/booking?' + new URLSearchParams({ ...stay, guests: String(stay.guests) }).toString()
}
