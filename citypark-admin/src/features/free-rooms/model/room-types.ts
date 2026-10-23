import type { components } from '../../../shared/api/schema'

export type RoomKey = keyof components['schemas']['HotelRooms']

export type RoomType = { key: RoomKey; label: string; total: number }

/** The five room types in display order. The totals match the maximums in the API contract. */
export const ROOM_TYPES: readonly RoomType[] = [
	{ key: 'vip', label: 'VIP-номер', total: 1 },
	{ key: 'singleBalcony', label: 'Одноместный с балконом', total: 1 },
	{ key: 'single', label: 'Одноместный', total: 2 },
	{ key: 'doubleTwin', label: 'Двухместный, 2 кровати', total: 4 },
	{ key: 'doubleEuro', label: 'Двухместный, кровать евро', total: 4 },
]

export const TOTAL_ROOMS = ROOM_TYPES.reduce((sum, t) => sum + t.total, 0)

const STALE_AFTER_MS = 24 * 60 * 60 * 1000

/** True when the sender's last report is older than 24 hours. */
export function isStale(updatedAt: string, now: number): boolean {
	return now - new Date(updatedAt).getTime() > STALE_AFTER_MS
}
