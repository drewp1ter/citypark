import type { AddressContext, AddressSurface, GuestAddress, SavedAddress } from '@/entities/address'
export type AddressEntry = GuestAddress | SavedAddress
export type ManagementProps = {
	context: AddressContext
	initialEntries: SavedAddress[]
	towns: readonly { name: string }[]
	suggestionsEnabled: boolean
	loadError?: string
	townsError?: boolean
}
export type ActionInput = { surface: AddressSurface; expectedAccountId: number; id?: number }
export type ManagementResult =
	| { ok: true; entry?: SavedAddress; created?: boolean }
	| { ok: false; message: string; fields?: Record<string, string>; authRequired?: boolean }
