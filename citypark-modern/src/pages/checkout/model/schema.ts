import { z } from 'zod'
import { isValidRuPhone } from '@/shared/lib'

export const methods = ['delivery', 'pickup'] as const
export const whens = ['asap', 'scheduled'] as const
export const zones = ['city', 'outskirts'] as const
export const payments = ['card', 'cash', 'sbp'] as const

const oneOf = <T extends readonly [string, ...string[]]>(values: T) => z.enum(values, { error: 'Выберите вариант' })

export const orderSchema = z.object({
	method: oneOf(methods),
	when: oneOf(whens),
	time: z.string(),
	zone: oneOf(zones),
	street: z.string().trim().max(120, 'Слишком длинный адрес'),
	apartment: z.string().trim().max(20, 'Не больше 20 символов'),
	entrance: z.string().trim().max(20, 'Не больше 20 символов'),
	name: z.string().trim().min(2, 'Укажите имя').max(80, 'Слишком длинное имя'),
	phone: z.string().refine(isValidRuPhone, 'Введите номер полностью: +7 ___ ___-__-__'),
	comment: z.string().trim().max(500, 'Не больше 500 символов'),
	payment: oneOf(payments),
	// The box is unchecked by default: consent to personal data processing must be given actively.
	consent: z.literal('on', { error: 'Нужно согласие с политикой конфиденциальности' }),
})

export type OrderField = keyof z.infer<typeof orderSchema>

/**
 * Rules that depend on other fields: an address for delivery, a time for "Ко времени…".
 * They are a separate function, not a refinement of the schema, because zod skips object-level refinements
 * while any field is invalid, and then the visitor would find these errors one submit later.
 */
export function conditionalErrors(order: Record<OrderField, string>): Partial<Record<OrderField, string>> {
	const errors: Partial<Record<OrderField, string>> = {}
	if (order.method === 'delivery' && order.street.trim().length < 3) {
		errors.street = 'Укажите улицу и дом'
	}
	if (order.when === 'scheduled') {
		if (!/^\d{2}:\d{2}$/.test(order.time)) errors.time = 'Выберите время'
		else if (order.time < '10:00' || order.time > '23:00') errors.time = 'Мы работаем с 10:00 до 23:00'
	}
	return errors
}

/** Lines of the cart as the browser sends them (ids and quantities only; prices come from the menu). */
export const linesSchema = z
	.array(z.object({ dishId: z.string(), variantId: z.string(), quantity: z.number().int().positive() }))
	.min(1, 'Корзина пуста')

/** "form" = an error that belongs to the whole order, not to one field. */
export type OrderErrorKey = OrderField | 'form'

export const orderFields = Object.keys(orderSchema.shape) as OrderField[]

/** What the form shows after a submit. On an error the typed values come back, so nothing is lost. */
export type OrderState =
	| { status: 'idle' }
	| { status: 'error'; errors: Partial<Record<OrderErrorKey, string>>; values: Record<OrderField, string> }
