import { z } from 'zod'

// Go's boundary trimming and rune limits use Unicode White_Space and code points.
const field = (min: number, max: number, short: string, long: string) =>
	z
		.string()
		.transform((value) => value.replace(/^\p{White_Space}+|\p{White_Space}+$/gu, ''))
		.refine((value) => [...value].length >= min, short)
		.refine((value) => [...value].length <= max, long)

export const addressFieldsSchema = z.object({
	name: field(0, 40, '', 'Не больше 40 символов'),
	city: field(1, 80, 'Выберите населённый пункт', 'Не больше 80 символов'),
	street: field(3, 120, 'Укажите улицу и дом', 'Слишком длинный адрес'),
	apartment: field(0, 20, '', 'Не больше 20 символов'),
	entrance: field(0, 20, '', 'Не больше 20 символов'),
})

export const guestBookSchema = z
	.object({
		version: z.literal(1),
		revision: z.uuid(),
		entries: z.array(addressFieldsSchema.extend({ id: z.uuid(), revision: z.uuid(), isDefault: z.boolean() }).strict()),
	})
	.strict()
	.refine((book) => new Set(book.entries.map((entry) => entry.id)).size === book.entries.length)
	.refine((book) => book.entries.filter((entry) => entry.isDefault).length <= 1)
