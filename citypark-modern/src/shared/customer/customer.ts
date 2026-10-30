import 'server-only'
import { serverEnv } from '@/shared/config'

export const customerFallback = 'Не удалось выполнить запрос. Попробуйте позже'

export class CustomerApiError extends Error {
	constructor(public readonly status: number, message: string, public readonly fields: Record<string, string> = {}) {
		super(message)
	}
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export async function customerRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
	if (!path.startsWith('/') || path.startsWith('//') || path.includes('\\')) throw new Error('Invalid customer API path')
	const response = await fetch(`${serverEnv().MENU_API_URL}${path}`, {
		...init,
		cache: 'no-store',
		headers: { 'Content-Type': 'application/json', ...init.headers },
	})
	const data: unknown = response.status === 204 ? undefined : await response.json().catch(() => undefined)
	if (!response.ok) {
		const fields: Record<string, string> = {}
		if (isRecord(data) && isRecord(data.errors)) {
			for (const [key, value] of Object.entries(data.errors)) if (typeof value === 'string') fields[key] = value
		}
		throw new CustomerApiError(response.status, isRecord(data) && typeof data.message === 'string' ? data.message : customerFallback, fields)
	}
	if (response.status !== 204 && !isRecord(data)) throw new CustomerApiError(502, customerFallback)
	return data as T
}

export type CustomerProfile = { id: number; name: string; email: string; phone: string; role: 'user' }
export type CustomerMethod = { provider: 'email' | 'google' | 'telegram'; verified: boolean }
export type CustomerPairing = { id: string; status: 'pending' | 'completed' | 'canceled'; sourceConfirmed: boolean; destinationConfirmed: boolean; otherName: string; expiresAt: string }

export async function customerAuthorized<T>(path: string, init: RequestInit = {}): Promise<T | undefined> {
 const { getCustomerSessionToken } = await import('./customer-session')
 const token = await getCustomerSessionToken()
 if (!token) return undefined
 return customerRequest<T>(path, { ...init, headers: { ...init.headers, Authorization: `Bearer ${token}` } })
}
