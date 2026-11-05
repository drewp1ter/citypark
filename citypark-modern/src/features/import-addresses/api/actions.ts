'use server'

import { revalidatePath } from 'next/cache'
import { addressFieldsSchema, type AddressSurface } from '@/entities/address'
import { createSavedAddress } from '@/entities/address/index.server'
import { CustomerApiError, customerRequest, getCustomerSessionToken, getMiniSessionToken } from '@/shared/customer'
import type { ImportResult } from '../model/import-state'

// Also used after client navigation: persistent layouts do not recheck cookies on their own.
export async function getImportAccountAction(surface: AddressSurface): Promise<number | null> {
	try {
		if (surface !== 'web' && surface !== 'mini') return null
		const token = await (surface === 'mini' ? getMiniSessionToken() : getCustomerSessionToken())
		if (!token) return null
		const profile = await customerRequest<{ id: number; role: string }>('/customer/me', {
			headers: { Authorization: `Bearer ${token}` },
		})
		return profile && profile.role === 'user' && Number.isSafeInteger(profile.id) && profile.id > 0 ? profile.id : null
	} catch {
		return null
	}
}

export async function importAddressAction(
	surface: AddressSurface,
	expectedAccountId: number,
	fields: unknown
): Promise<ImportResult> {
	if ((surface !== 'web' && surface !== 'mini') || !Number.isSafeInteger(expectedAccountId) || expectedAccountId < 1)
		return { ok: false, kind: 'auth', message: 'Войдите повторно, чтобы добавить адреса' }
	const parsed = addressFieldsSchema.strict().safeParse(fields)
	if (!parsed.success) {
		const errors: Record<string, string> = {}
		for (const issue of parsed.error.issues) if (issue.path.length) errors[String(issue.path[0])] ??= issue.message
		return { ok: false, kind: 'validation', message: 'Проверьте адрес', fields: errors }
	}
	try {
		// The entity validates the response and freshly verifies this surface's session/profile.
		const result = await createSavedAddress(surface, expectedAccountId, parsed.data)
		revalidatePath(surface === 'web' ? '/account/addresses' : '/mini/addresses')
		return { ok: true, ...result }
	} catch (error) {
		if (error instanceof CustomerApiError && error.status < 500)
			return {
				ok: false,
				kind: error.status === 401 ? 'auth' : 'validation',
				message: error.message,
				fields: error.fields,
			}
		return { ok: false, kind: 'request', message: 'Не удалось добавить адреса. Попробуйте ещё раз.' }
	}
}
