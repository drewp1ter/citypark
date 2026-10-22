import type { Room } from './types'

/** Lowest price per night of the given rooms, or of all rooms when `ids` is omitted. `null` for an empty list. */
export function minPricePerNight(rooms: readonly Room[], ids?: readonly string[]): number | null {
	const prices = rooms.filter((room) => !ids || ids.includes(room.id)).map((room) => room.pricePerNight)
	return prices.length > 0 ? Math.min(...prices) : null
}

/** True when the given rooms are more than one and not all the same price, so the price reads "от …". */
export function hasPriceRange(rooms: readonly Room[], ids: readonly string[]): boolean {
	const prices = new Set(rooms.filter((room) => ids.includes(room.id)).map((room) => room.pricePerNight))
	return prices.size > 1
}
