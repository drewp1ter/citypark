'use server'

import { redirect } from 'next/navigation'
import { revalidatePath } from 'next/cache'
import { customerAuthorized, type CustomerProfile } from '@/shared/customer'
import { profileSchema } from '../model/schema'
import { customerRequest, CustomerApiError, customerFallback, clearCustomerSession, getCustomerSessionToken, setCustomerSession } from '@/shared/customer'
import { emailSchema, registerSchema, signInSchema, resetSchema, type AuthState } from '../model/schema'

function valuesOf(form: FormData, keys: string[]): Record<string, string> {
	return Object.fromEntries(keys.map((key) => [key, String(form.get(key) ?? '')]))
}

function errorsOf(error: { issues: { path: PropertyKey[]; message: string }[] }): Record<string, string> {
	return Object.fromEntries(error.issues.map((issue) => [String(issue.path[0]), issue.message]))
}

function failed(error: unknown, values: Record<string, string>): AuthState {
	return { status: 'error', message: error instanceof CustomerApiError ? error.message : customerFallback, fields: error instanceof CustomerApiError ? error.fields : {}, values }
}

function safeReturnPath(raw: string): string {
	if (!raw.startsWith('/') || raw.startsWith('//') || raw.includes('\\') || /[\u0000-\u001f\u007f]/.test(raw)) return '/menu'
	if (/%(?:2f|5c|00|0[0-9a-f]|1[0-9a-f]|7f)/i.test(raw)) return '/menu'
	try {
		const parsed = new URL(raw, 'http://local.invalid')
		if (parsed.origin !== 'http://local.invalid' || parsed.pathname.startsWith('//')) return '/menu'
		return `${parsed.pathname}${parsed.search}${parsed.hash}`
	} catch { return '/menu' }
}

export async function signInAction(_previous: AuthState, form: FormData): Promise<AuthState> {
	const values = valuesOf(form, ['email'])
	const parsed = signInSchema.safeParse({ email: values.email, password: String(form.get('password') ?? '') })
	if (!parsed.success) return { status: 'error', fields: errorsOf(parsed.error), values }
	let destination: string
	try {
		const result = await customerRequest<{ token: string; expiresAt: string }>('/auth/customer/sign-in', { method: 'POST', body: JSON.stringify(parsed.data) })
		await setCustomerSession(result.token, result.expiresAt)
		destination = safeReturnPath(String(form.get('returnTo') ?? ''))
	} catch (error) { return failed(error, values) }
	redirect(destination)
}

export async function registerAction(_previous: AuthState, form: FormData): Promise<AuthState> {
	const values = valuesOf(form, ['name', 'email', 'phone'])
	const parsed = registerSchema.safeParse({ ...values, password: String(form.get('password') ?? '') })
	if (!parsed.success) return { status: 'error', fields: errorsOf(parsed.error), values }
	try {
		const result = await customerRequest<{ message: string }>('/auth/customer/register', { method: 'POST', body: JSON.stringify(parsed.data) })
		return { status: 'success', message: result.message }
	} catch (error) { return failed(error, values) }
}

export async function resendAction(_previous: AuthState, form: FormData): Promise<AuthState> {
	const values = valuesOf(form, ['email'])
	const parsed = emailSchema.safeParse(values.email)
	if (!parsed.success) return { status: 'error', fields: { email: parsed.error.issues[0].message }, values }
	try {
		const result = await customerRequest<{ message: string }>('/auth/customer/verification/resend', { method: 'POST', body: JSON.stringify({ email: parsed.data }) })
		return { status: 'success', message: result.message }
	} catch (error) { return failed(error, values) }
}

export async function verifyAction(_previous: AuthState, form: FormData): Promise<AuthState> {
	const token = String(form.get('token') ?? '')
	if (!token || token.length > 512) return { status: 'error', message: 'Ссылка недействительна или устарела' }
	try {
		await customerRequest<{ verified: boolean }>('/auth/customer/verification/confirm', { method: 'POST', body: JSON.stringify({ token }) })
		return { status: 'success', message: 'Email подтверждён. Теперь войдите в аккаунт.' }
	} catch (error) { return failed(error, {}) }
}

export async function recoveryAction(_previous: AuthState, form: FormData): Promise<AuthState> {
	const values = valuesOf(form, ['email'])
	const parsed = emailSchema.safeParse(values.email)
	if (!parsed.success) return { status: 'error', fields: { email: parsed.error.issues[0].message }, values }
	try {
		const result = await customerRequest<{ message: string }>('/auth/customer/password/forgot', { method: 'POST', body: JSON.stringify({ email: parsed.data }) })
		return { status: 'success', message: result.message }
	} catch (error) { return failed(error, values) }
}

export async function resetAction(_previous: AuthState, form: FormData): Promise<AuthState> {
	const parsed = resetSchema.safeParse({ token: String(form.get('token') ?? ''), password: String(form.get('password') ?? '') })
	if (!parsed.success) return { status: 'error', fields: errorsOf(parsed.error), message: parsed.error.issues.some((issue) => issue.path[0] === 'token') ? 'Ссылка недействительна или устарела' : undefined }
	try {
		await customerRequest<void>('/auth/customer/password/reset', { method: 'POST', body: JSON.stringify(parsed.data) })
		await clearCustomerSession()
		return { status: 'success', message: 'Пароль изменён. Войдите с новым паролем' }
	} catch (error) { return failed(error, {}) }
}

export async function confirmEmailAction(_previous: AuthState, form: FormData): Promise<AuthState> {
	const token = String(form.get('token') ?? '')
	if (!token || token.length > 512) return { status: 'error', message: 'Ссылка недействительна или устарела' }
	const session = await getCustomerSessionToken()
	if (!session) return { status: 'error', message: 'Войдите повторно для подтверждения email' }
	try {
		await customerRequest<{ verified: boolean }>('/customer/email/confirm', { method: 'POST', headers: { Authorization: `Bearer ${session}` }, body: JSON.stringify({ token }) })
		return { status: 'success', message: 'Новый email подтверждён' }
	} catch (error) {
		if (error instanceof CustomerApiError && (error.status === 401 || error.status === 403)) return { status: 'error', message: 'Войдите повторно для подтверждения email' }
		return failed(error, {})
	}
}

export async function logoutAction(): Promise<void> {
	const token = await getCustomerSessionToken()
	let failedRemote = false
	if (token) {
		try { await customerRequest<void>('/auth/customer/logout', { method: 'POST', headers: { Authorization: `Bearer ${token}` } }) }
		catch { failedRemote = true }
	}
	await clearCustomerSession()
	redirect(failedRemote ? '/account/sign-in?logout=remote-failed' : '/account/sign-in?logout=1')
}

export async function saveProfileAction(_previous: AuthState, form: FormData): Promise<AuthState> {
 const values = valuesOf(form, ['name', 'phone'])
 const parsed = profileSchema.safeParse(values)
 if (!parsed.success) return { status: 'error', fields: errorsOf(parsed.error), values }
 try {
  const updated = await customerAuthorized<CustomerProfile>('/customer/me', { method: 'PUT', body: JSON.stringify(parsed.data) })
  if (!updated) return { status: 'error', message: 'Войдите повторно', values }
  revalidatePath('/account')
  return { status: 'success', message: 'Профиль сохранён', values: { name: updated.name, phone: updated.phone } }
 } catch (error) { return failed(error, values) }
}

export async function changeEmailAction(_previous: AuthState, form: FormData): Promise<AuthState> {
 const values = valuesOf(form, ['email'])
 const parsed = emailSchema.safeParse(values.email)
 if (!parsed.success) return { status: 'error', fields: { email: parsed.error.issues[0].message }, values }
 try {
  const result = await customerAuthorized<{ message: string }>('/customer/email/change', { method: 'POST', body: JSON.stringify({ email: parsed.data }) })
  if (!result) return { status: 'error', message: 'Войдите повторно', values }
  return { status: 'success', message: result.message }
 } catch (error) { return failed(error, values) }
}
