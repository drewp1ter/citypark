import 'server-only'
import { CustomerApiError, customerRequest, getCustomerSessionToken, getMiniSessionToken } from '@/shared/customer'
import {
	addressCreatedSchema,
	addressEntrySchema,
	addressFieldsSchema,
	addressIdSchema,
	addressListSchema,
	customerProfileSchema,
} from '../model/schema'
import type { AddressBookLoad, AddressFields, AddressSurface, SavedAddress } from '../model/types'
export type { AddressContext, AddressBookLoad } from '../model/types'

const loadError = 'Не удалось загрузить адреса. Попробуйте ещё раз.'
const saveError = 'Не удалось сохранить адрес. Попробуйте ещё раз.'
const signIn = 'Войдите повторно, чтобы сохранить адрес'

async function session(surface: AddressSurface) {
	if (surface === 'web') return getCustomerSessionToken()
	if (surface === 'mini') return getMiniSessionToken()
	throw new Error('Invalid address surface')
}
const authorization = (token: string) => ({ Authorization: `Bearer ${token}` })
async function profile(token: string) {
	return customerProfileSchema.parse(await customerRequest<unknown>('/customer/me', { headers: authorization(token) }))
}

export async function loadAddressBook(surface: AddressSurface): Promise<AddressBookLoad> {
	let accountId: number | undefined
	try {
		const token = await session(surface)
		if (!token)
			return {
				context: surface === 'web' ? { kind: 'guest', surface } : { kind: 'auth-required', surface },
				entries: [],
			}
		accountId = (await profile(token)).id
		const data = addressListSchema.parse(
			await customerRequest<unknown>('/customer/addresses', { headers: authorization(token) })
		)
		return { context: { kind: 'account', surface, accountId }, entries: data.addresses }
	} catch (error) {
		if (error instanceof CustomerApiError && error.status === 401)
			return { context: { kind: 'auth-required', surface }, entries: [] }
		return {
			context: accountId === undefined ? { kind: 'auth-required', surface } : { kind: 'account', surface, accountId },
			entries: [],
			error: loadError,
		}
	}
}

// The expected account only guards stale UI; the backend derives ownership from this session.
async function ownerToken(surface: AddressSurface, expectedAccountId: number) {
	const token = await session(surface)
	if (!token || !addressIdSchema.safeParse(expectedAccountId).success) throw new CustomerApiError(401, signIn)
	if ((await profile(token)).id !== expectedAccountId) throw new CustomerApiError(401, signIn)
	return token
}
const knownMessages: Record<number, readonly string[]> = {
	400: ['Некорректный JSON', 'Проверьте адрес'],
	403: ['Недостаточно прав'],
	404: ['Не найдено'],
	409: ['Такой адрес уже сохранён'],
	503: ['Вход временно недоступен'],
}
function failure(error: unknown): never {
	if (error instanceof CustomerApiError) {
		if (error.status === 401) throw new CustomerApiError(401, signIn)
		if (knownMessages[error.status]?.includes(error.message)) {
			const fields: Record<string, string> = {}
			for (const key of ['name', 'city', 'street', 'apartment', 'entrance']) {
				const message = error.fields[key]
				if (
					[
						'Не больше 40 символов',
						'Выберите населённый пункт',
						'Не больше 80 символов',
						'Укажите улицу и дом',
						'Слишком длинный адрес',
						'Не больше 20 символов',
						'Такой адрес уже сохранён',
					].includes(message)
				)
					fields[key] = message
			}
			throw new CustomerApiError(error.status, error.message, fields)
		}
	}
	throw new CustomerApiError(502, saveError)
}
function fieldsBody(fields: AddressFields) {
	const parsed = addressFieldsSchema.safeParse(fields)
	if (!parsed.success) {
		const errors: Record<string, string> = {}
		for (const issue of parsed.error.issues) if (issue.path.length) errors[String(issue.path[0])] ??= issue.message
		throw new CustomerApiError(400, 'Проверьте адрес', errors)
	}
	return JSON.stringify(parsed.data)
}
function entryId(id: number) {
	if (!addressIdSchema.safeParse(id).success) throw new CustomerApiError(404, 'Не найдено')
	return id
}
export async function createSavedAddress(
	surface: AddressSurface,
	expectedAccountId: number,
	fields: AddressFields
): Promise<{ entry: SavedAddress; created: boolean }> {
	try {
		const token = await ownerToken(surface, expectedAccountId)
		return addressCreatedSchema.parse(
			await customerRequest<unknown>('/customer/addresses', {
				method: 'POST',
				headers: authorization(token),
				body: fieldsBody(fields),
			})
		)
	} catch (error) {
		return failure(error)
	}
}
export async function updateSavedAddress(
	surface: AddressSurface,
	expectedAccountId: number,
	id: number,
	fields: AddressFields
): Promise<SavedAddress> {
	try {
		const token = await ownerToken(surface, expectedAccountId)
		return addressEntrySchema.parse(
			await customerRequest<unknown>(`/customer/addresses/${entryId(id)}`, {
				method: 'PUT',
				headers: authorization(token),
				body: fieldsBody(fields),
			})
		).entry
	} catch (error) {
		return failure(error)
	}
}
export async function setSavedDefault(
	surface: AddressSurface,
	expectedAccountId: number,
	id: number
): Promise<SavedAddress> {
	try {
		const token = await ownerToken(surface, expectedAccountId)
		return addressEntrySchema.parse(
			await customerRequest<unknown>(`/customer/addresses/${entryId(id)}/default`, {
				method: 'PUT',
				headers: authorization(token),
			})
		).entry
	} catch (error) {
		return failure(error)
	}
}
export async function deleteSavedAddress(
	surface: AddressSurface,
	expectedAccountId: number,
	id: number
): Promise<void> {
	try {
		const token = await ownerToken(surface, expectedAccountId)
		const result = await customerRequest<unknown>(`/customer/addresses/${entryId(id)}`, {
			method: 'DELETE',
			headers: authorization(token),
		})
		if (result !== undefined) throw new Error('Invalid delete response')
	} catch (error) {
		failure(error)
	}
}
