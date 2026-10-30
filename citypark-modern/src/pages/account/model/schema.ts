import { z } from 'zod'

export const emailSchema = z.email('Укажите корректный email').max(254, 'Слишком длинный email')
export const signInSchema = z.object({
	email: emailSchema,
	password: z.string().min(1, 'Введите пароль'),
})
export const registerSchema = signInSchema.extend({
	name: z.string().trim().min(1, 'Введите имя').max(80, 'Имя слишком длинное'),
	password: z.string().min(8, 'Пароль должен содержать не менее 8 символов').max(64, 'Пароль слишком длинный'),
	phone: z.string().trim().optional(),
})
export const resetSchema = z.object({ token: z.string().min(1).max(512), password: z.string().min(8, 'Пароль должен содержать не менее 8 символов').max(64, 'Пароль слишком длинный') })

export type AuthState = { status: 'idle' | 'error' | 'success'; message?: string; fields?: Record<string, string>; values?: Record<string, string> }
export const initialAuthState: AuthState = { status: 'idle' }
