import { z } from 'zod'
import { bookingUuid, type BookingStay, type BookingQuote, type BookingReservation } from '@/entities/room/index.client'
const noControls = (value: string) => !/\p{Cc}/u.test(value)
export const guestSchema = z
	.object({
		name: z
			.string()
			.refine(noControls, 'Укажите имя без управляющих символов')
			.transform((v) => v.trim())
			.refine((v) => [...v].length >= 1 && [...v].length <= 150, 'Укажите имя (до 150 символов)'),
		email: z
			.string()
			.refine(noControls, 'Проверьте электронную почту')
			.transform((v) => v.trim())
			.pipe(z.email('Проверьте электронную почту'))
			.refine((v) => new TextEncoder().encode(v).length <= 254, 'Слишком длинный адрес')
			.transform((v) => {
				const at = v.lastIndexOf('@')
				return v.slice(0, at + 1) + v.slice(at + 1).toLowerCase()
			}),
		phone: z
			.string()
			.refine(noControls, 'Проверьте телефон')
			.transform((v) => {
				let phone = v.replace(/[ ()-]/g, '')
				if (/^8[0-9]{10}$/.test(phone)) phone = '+7' + phone.slice(1)
				else if (/^[0-9]{8,15}$/.test(phone)) phone = '+' + phone
				return phone
			})
			.pipe(z.string().regex(/^\+[0-9]{8,15}$/, 'Укажите телефон: от 8 до 15 цифр')),
	})
	.strict()
export const createSchema = z.object({ idempotencyKey: bookingUuid, quoteId: bookingUuid, guest: guestSchema }).strict()
export type BookingGuest = z.infer<typeof guestSchema>
export type BookingAttempt = z.infer<typeof createSchema>
export type BookingValues = BookingStay & { guest: BookingGuest }
export type BookingState = {
	status: 'idle' | 'quoted' | 'error' | 'pending' | 'unknown' | 'confirmed'
	message?: string
	code?: string
	errors?: Record<string, string>
	values?: BookingValues
	quote?: BookingQuote
	idempotencyKey?: string
	reservation?: BookingReservation
}
