'use server'

import { revalidatePath } from 'next/cache'
import { addressFieldsSchema } from '@/entities/address'
import { createSavedAddress, updateSavedAddress, deleteSavedAddress, setSavedDefault } from '@/entities/address/index.server'
import { CustomerApiError } from '@/shared/customer'
import type { ActionInput, ManagementResult } from '../model/types'

function valid(input: ActionInput, requireId = false) {
	return (
		input &&
		(input.surface === 'web' || input.surface === 'mini') &&
		Number.isSafeInteger(input.expectedAccountId) &&
		input.expectedAccountId > 0 &&
		(input.id === undefined ? !requireId : Number.isSafeInteger(input.id) && input.id > 0)
	)
}
function invalidate(input: ActionInput) {
	revalidatePath(input.surface === 'web' ? '/account/addresses' : '/mini/addresses')
	revalidatePath(input.surface === 'web' ? '/checkout' : '/mini/checkout')
}
function failure(error: unknown): ManagementResult {
	if (error instanceof CustomerApiError)
		return { ok: false, message: error.message, fields: error.fields, authRequired: error.status === 401 }
	return { ok: false, message: 'Не удалось сохранить адрес. Попробуйте ещё раз.' }
}
export async function saveAddressAction(input: ActionInput, fields: unknown): Promise<ManagementResult> {
	if (!valid(input)) return { ok: false, message: 'Войдите повторно, чтобы сохранить адрес', authRequired: true }
	const parsed = addressFieldsSchema.safeParse(fields)
	if (!parsed.success) {
		const errors: Record<string, string> = {}
		for (const issue of parsed.error.issues) errors[String(issue.path[0])] ??= issue.message
		return { ok: false, message: 'Проверьте адрес', fields: errors }
	}
	try {
		const result =
			input.id === undefined
				? await createSavedAddress(input.surface, input.expectedAccountId, parsed.data)
				: {
						entry: await updateSavedAddress(input.surface, input.expectedAccountId, input.id, parsed.data),
						created: false,
					}
		invalidate(input)
		return { ok: true, ...result }
	} catch (error) {
		return failure(error)
	}
}
export async function deleteAddressAction(input: ActionInput): Promise<ManagementResult> {
	if (!valid(input, true)) return { ok: false, message: 'Войдите повторно, чтобы сохранить адрес', authRequired: true }
	try {
		await deleteSavedAddress(input.surface, input.expectedAccountId, input.id!)
		invalidate(input)
		return { ok: true }
	} catch (error) {
		return failure(error)
	}
}
export async function defaultAddressAction(input: ActionInput): Promise<ManagementResult> {
	if (!valid(input, true)) return { ok: false, message: 'Войдите повторно, чтобы сохранить адрес', authRequired: true }
	try {
		const entry = await setSavedDefault(input.surface, input.expectedAccountId, input.id!)
		invalidate(input)
		return { ok: true, entry }
	} catch (error) {
		return failure(error)
	}
}
