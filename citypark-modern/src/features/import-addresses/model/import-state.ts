import type {
	AddressFields,
	AddressSurface,
	GuestBook,
	GuestResult,
	GuestStatus,
	SavedAddress,
} from '@/entities/address'

export type ImportResult =
	| { ok: true; entry: SavedAddress; created: boolean }
	| { ok: false; kind: 'auth' | 'validation' | 'request'; message: string; fields?: Record<string, string> }
type GuestSnapshot = { status: GuestStatus; book: GuestBook }
export type ImportState = {
	accountId: number | null
	open: boolean
	phase: 'idle' | 'offered' | 'pending' | 'error' | 'success'
	created: number
	skipped: number
	needsReview: boolean
	message: string
	focus: number
}
const empty: ImportState = {
	accountId: null,
	open: false,
	phase: 'idle',
	created: 0,
	skipped: 0,
	needsReview: false,
	message: '',
	focus: 0,
}
const requestError = 'Не удалось добавить адреса. Попробуйте ещё раз.'

type Dependencies = {
	guest: () => GuestSnapshot
	verify: (surface: AddressSurface) => Promise<number | null>
	importOne: (surface: AddressSurface, expectedAccountId: number, fields: AddressFields) => Promise<ImportResult>
	remove: (acknowledged: readonly { id: string; revision: string }[]) => Promise<GuestResult>
	storage: () => Pick<Storage, 'getItem' | 'setItem'>
}

// Each surface has one browser coordinator shared by its layout offer and management trigger.
// Only dismissal keys survive a context change; responses and queued work belong to an epoch.
export function createImportCoordinator(surface: AddressSurface, deps: Dependencies) {
	let state = empty
	let epoch = 0
	let syncEpoch = 0
	let lease: object | null = null
	const listeners = new Set<() => void>()
	const dismissed = new Set<string>()
	const emit = (patch: Partial<ImportState>) => {
		state = { ...state, ...patch }
		listeners.forEach((listener) => listener())
	}
	const reset = (accountId: number | null) => {
		epoch++
		emit({ ...empty, accountId })
	}
	const key = (revision: string) => `citypark.address-import:${surface}:${state.accountId}:${revision}`
	function suppressed(revision: string) {
		const marker = key(revision)
		if (dismissed.has(marker)) return true
		try {
			return deps.storage().getItem(marker) === 'dismissed'
		} catch {
			return false
		}
	}
	function offer(manual = false) {
		const guest = deps.guest()
		if (!state.accountId || guest.status !== 'ready' || !guest.book.entries.length) return
		if (!manual && (state.open || suppressed(guest.book.revision))) return
		if (state.phase === 'pending') return
		emit({
			open: true,
			phase: 'offered',
			created: 0,
			skipped: 0,
			needsReview: false,
			message: '',
			focus: manual ? state.focus + 1 : state.focus,
		})
	}
	function dismiss() {
		if (!state.accountId) return
		const marker = key(deps.guest().book.revision)
		dismissed.add(marker)
		try {
			deps.storage().setItem(marker, 'dismissed')
		} catch {
			/* The memory marker still prevents duplicate prompts. */
		}
		epoch++
		emit({ open: false, phase: 'idle', created: 0, skipped: 0, needsReview: false, message: '' })
	}
	async function synchronize() {
		const current = ++syncEpoch
		reset(null)
		let accountId: number | null = null
		try {
			accountId = await deps.verify(surface)
		} catch {
			/* Optional authentication failure never blocks the page. */
		}
		if (current !== syncEpoch || !lease) return
		reset(accountId)
		offer()
		return accountId
	}
	async function start(changed: () => void) {
		if (!state.accountId || !state.open || state.phase === 'pending') return
		const guest = deps.guest()
		if (guest.status !== 'ready') return
		const accountId = state.accountId
		const run = ++epoch
		const valid = () => run === epoch && state.accountId === accountId && lease !== null
		const snapshot = [
			...guest.book.entries.filter((entry) => entry.isDefault),
			...guest.book.entries.filter((entry) => !entry.isDefault),
		].map((entry) => ({ ...entry }))
		emit({ phase: 'pending', created: 0, skipped: 0, needsReview: false, message: '' })
		async function currentAccount() {
			const id = await deps.verify(surface)
			if (!valid()) return false
			if (id === null) {
				emit({ phase: 'error', message: 'Войдите повторно, чтобы добавить адреса' })
				return false
			}
			if (id !== accountId) {
				reset(id)
				offer()
				return false
			}
			return true
		}
		try {
			for (const entry of snapshot) {
				if (!valid() || !(await currentAccount()) || !valid()) return
				const { name, city, street, apartment, entrance } = entry
				const result = await deps.importOne(surface, accountId, { name, city, street, apartment, entrance })
				if (!valid()) return
				if (!result.ok) {
					emit({ phase: 'error', message: [result.message, ...Object.values(result.fields ?? {})].join(' ') })
					return
				}
				// A committed request cannot be undone by logout. A late response is never
				// permission to clean up this browser's data under a changed session.
				if (!(await currentAccount()) || !valid()) return
				emit({
					created: state.created + (result.created ? 1 : 0),
					skipped: state.skipped + (result.created ? 0 : 1),
					needsReview: state.needsReview || result.entry.needsReview,
				})
				changed()
				if (!valid()) return
				const cleanup = await deps.remove([{ id: entry.id, revision: entry.revision }])
				if (!valid()) return
				if (!cleanup.ok) {
					emit({ phase: 'error', message: cleanup.message })
					return
				}
			}
			if (valid()) emit({ phase: 'success' })
		} catch {
			if (valid()) emit({ phase: 'error', message: requestError })
		}
	}
	return {
		subscribe(listener: () => void) {
			listeners.add(listener)
			return () => {
				listeners.delete(listener)
			}
		},
		getSnapshot: () => state,
		getServerSnapshot: () => empty,
		attach() {
			syncEpoch++
			reset(null)
			const owner = {}
			lease = owner
			return () => {
				if (lease !== owner) return
				lease = null
				syncEpoch++
				reset(null)
			}
		},
		synchronize,
		offer,
		dismiss,
		start,
	}
}
