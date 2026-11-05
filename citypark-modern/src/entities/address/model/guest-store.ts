import { addressFieldsSchema, guestBookSchema } from './schema'
import { addressDuplicateKey, formatAddress } from './format'
import type { AddressFields, GuestAddress, GuestBook, GuestResult, GuestStatus } from './types'

const KEY = 'citypark.addresses.v1'
const EMPTY: GuestBook = { version: 1, revision: '', entries: [] }
const SERVER = { status: 'loading' as GuestStatus, book: EMPTY }
const messages = {
	corrupt: 'Не удалось прочитать сохранённые адреса',
	unsupported: 'Обновите страницу, чтобы открыть сохранённые адреса',
	unavailable: 'Не удалось сохранить адрес в этом браузере',
	missing: 'Адрес больше недоступен',
}
let cachedRaw: string | null | undefined
let snapshot = SERVER
const listeners = new Set<() => void>()

export function getServerGuestSnapshot() {
	return SERVER
}

export function getGuestSnapshot(): typeof SERVER {
	if (typeof window === 'undefined') return SERVER
	let raw: string | null
	try {
		raw = window.localStorage.getItem(KEY)
	} catch {
		if (snapshot.status !== 'unavailable') snapshot = { status: 'unavailable', book: EMPTY }
		cachedRaw = undefined
		return snapshot
	}
	if (raw === cachedRaw) return snapshot
	cachedRaw = raw
	if (raw === null) snapshot = { status: 'ready', book: EMPTY }
	else {
		try {
			const value: unknown = JSON.parse(raw)
			if (value && typeof value === 'object' && 'version' in value && value.version !== 1) {
				snapshot = { status: 'unsupported', book: EMPTY }
			} else {
				const parsed = guestBookSchema.safeParse(value)
				snapshot = parsed.success ? { status: 'ready', book: parsed.data } : { status: 'corrupt', book: EMPTY }
			}
		} catch {
			snapshot = { status: 'corrupt', book: EMPTY }
		}
	}
	return snapshot
}

export function subscribeToGuestAddresses(listener: () => void): () => void {
	listeners.add(listener)
	const onStorage = (event: StorageEvent) => {
		if (event.key === KEY || event.key === null) {
			getGuestSnapshot()
			listener()
		}
	}
	window.addEventListener('storage', onStorage)
	return () => {
		listeners.delete(listener)
		window.removeEventListener('storage', onStorage)
	}
}

// Each operation re-reads storage. This preserves sequential tab edits, but localStorage
// cannot provide an atomic transaction across simultaneous writers in different tabs.
async function mutate(change: (book: GuestBook) => GuestResult, reset = false): Promise<GuestResult> {
	const current = getGuestSnapshot()
	if (reset ? current.status !== 'corrupt' : current.status !== 'ready') {
		return {
			ok: false,
			message:
				messages[
					current.status === 'unsupported' ? 'unsupported' : current.status === 'corrupt' ? 'corrupt' : 'unavailable'
				],
		}
	}
	try {
		const book: GuestBook = {
			version: 1,
			revision: current.book.revision,
			entries: current.book.entries.map((entry) => ({ ...entry })),
		}
		const result = change(book)
		if (!result.ok) return result
		if (!reset && JSON.stringify(book) === JSON.stringify(current.book)) return result
		book.revision = crypto.randomUUID()
		window.localStorage.setItem(KEY, JSON.stringify(book))
		getGuestSnapshot()
		listeners.forEach((listener) => listener())
		return result
	} catch {
		return { ok: false, message: messages.unavailable }
	}
}

export async function saveGuestAddress(fields: AddressFields, id?: string): Promise<GuestResult> {
	return mutate((book) => {
		const parsed = addressFieldsSchema.safeParse(fields)
		if (!parsed.success) return { ok: false, message: parsed.error.issues[0].message }
		const existing = id === undefined ? undefined : book.entries.find((entry) => entry.id === id)
		if (id !== undefined && !existing) return { ok: false, message: messages.missing }
		const key = addressDuplicateKey(parsed.data.city, formatAddress(parsed.data))
		const duplicate = book.entries.find(
			(entry) => entry.id !== id && addressDuplicateKey(entry.city, formatAddress(entry)) === key
		)
		if (duplicate)
			return id === undefined
				? { ok: true, entry: { ...duplicate }, created: false }
				: { ok: false, message: 'Такой адрес уже сохранён' }
		if (existing) {
			if (Object.entries(parsed.data).some(([key, value]) => existing[key as keyof AddressFields] !== value)) {
				Object.assign(existing, parsed.data, { revision: crypto.randomUUID() })
			}
			return { ok: true, entry: { ...existing }, created: false }
		}
		const entry: GuestAddress = {
			...parsed.data,
			id: crypto.randomUUID(),
			revision: crypto.randomUUID(),
			isDefault: book.entries.length === 0,
		}
		book.entries.push(entry)
		return { ok: true, entry: { ...entry }, created: true }
	})
}

export async function deleteGuestAddress(id: string): Promise<GuestResult> {
	return mutate((book) => {
		book.entries = book.entries.filter((entry) => entry.id !== id)
		return { ok: true }
	})
}

export async function setGuestDefault(id: string): Promise<GuestResult> {
	return mutate((book) => {
		if (!book.entries.some((entry) => entry.id === id)) return { ok: false, message: messages.missing }
		for (const entry of book.entries) {
			if (entry.isDefault !== (entry.id === id)) {
				entry.isDefault = entry.id === id
				entry.revision = crypto.randomUUID()
			}
		}
		return { ok: true }
	})
}

export async function removeImportedGuests(
	acknowledged: readonly { id: string; revision: string }[]
): Promise<GuestResult> {
	return mutate((book) => {
		book.entries = book.entries.filter(
			(entry) => !acknowledged.some((ack) => ack.id === entry.id && ack.revision === entry.revision)
		)
		return { ok: true }
	})
}

/** Call only after the user confirms «Сбросить сохранённые адреса». */
export async function resetGuestAddresses(): Promise<GuestResult> {
	return mutate((book) => {
		book.entries = []
		return { ok: true }
	}, true)
}
