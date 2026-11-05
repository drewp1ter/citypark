export type AddressFields = { name: string; city: string; street: string; apartment: string; entrance: string }
export type SavedAddress = AddressFields & {
	id: number
	address: string
	isDefault: boolean
	needsReview: boolean
	createdAt: string
	updatedAt: string
}
export type GuestAddress = AddressFields & { id: string; revision: string; isDefault: boolean }
export type GuestBook = { version: 1; revision: string; entries: GuestAddress[] }
export type AddressSurface = 'web' | 'mini'
export type GuestStatus = 'loading' | 'ready' | 'corrupt' | 'unsupported' | 'unavailable'
export type GuestResult = { ok: true; entry?: GuestAddress; created?: boolean } | { ok: false; message: string }

export type AddressContext =
	| { kind: 'guest'; surface: 'web' }
	| { kind: 'account'; surface: AddressSurface; accountId: number }
	| { kind: 'auth-required'; surface: AddressSurface }
export type AddressBookLoad = { context: AddressContext; entries: SavedAddress[]; error?: string }
