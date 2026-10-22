import { z } from 'zod'
import { isValidRuPhone } from '@/shared/lib'

export const topics = [
	{ value: 'hotel', label: 'Отель' },
	{ value: 'banquet', label: 'Банкет' },
	{ value: 'delivery', label: 'Доставка' },
	{ value: 'review', label: 'Отзыв' },
] as const

const topicValues = topics.map((topic) => topic.value) as [string, ...string[]]

export const contactSchema = z.object({
	name: z.string().trim().min(2, 'Укажите имя').max(80, 'Слишком длинное имя'),
	// One field for either a phone number or an e-mail address.
	contact: z
		.string()
		.trim()
		.refine((value) => isValidRuPhone(value) || z.email().safeParse(value).success, 'Укажите телефон или e-mail'),
	topic: z.enum(topicValues, { error: 'Выберите тему' }),
	message: z.string().trim().min(5, 'Напишите хотя бы несколько слов').max(2000, 'Не больше 2000 символов'),
})

export type ContactField = keyof z.infer<typeof contactSchema>

export const contactFields = Object.keys(contactSchema.shape) as ContactField[]

/** What the form shows after each submit. On an error the typed values come back, so nothing is lost. */
export type ContactState =
	| { status: 'idle' }
	| { status: 'success' }
	| { status: 'error'; errors: Partial<Record<ContactField, string>>; values: Record<ContactField, string> }
