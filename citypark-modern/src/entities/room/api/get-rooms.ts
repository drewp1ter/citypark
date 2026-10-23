import { todayInMoscow, typo } from '@/shared/lib'
import { fetchHotelRooms } from './hotel-api'
import { fetchRooms } from './rooms-api'
import type { Room, RoomAvailability, StayQuery } from '../model/types'

// Reads go through these functions only. Rooms come from the backend; availability will later come from a booking engine.

/** The room types in the backend's order. Errors are left to the caller. */
export async function getRooms(): Promise<readonly Room[]> {
	return (await fetchRooms()).map((room) => ({
		id: room.id,
		name: room.name,
		hotelName: room.hotelName || undefined,
		kind: room.kind,
		balcony: room.balcony,
		numbers: typo(room.numbers),
		area: typo(room.area),
		areaDetailed: typo(room.areaDetailed) || undefined,
		capacity: room.capacity || undefined,
		descriptionHtml: room.description,
		summary: typo(room.summary) || undefined,
		pricePerNight: room.pricePerNight,
		photo: { src: room.photo.url, alt: room.photo.alt, position: room.photo.position || undefined },
		gallery: (room.gallery ?? []).map((photo) => ({ src: photo.url, alt: photo.alt })),
		hotelNote: typo(room.hotelNote),
	}))
}

/** One room type by id, or `undefined` when there is none. Errors are left to the caller. */
export async function getRoom(id: string): Promise<Room | undefined> {
	return (await getRooms()).find((room) => room.id === id)
}

// Website room id → key in `GET /hotel/rooms` → how many rooms of that type the hotel has.
const roomTypes = [
	{ id: 'vip', key: 'vip', total: 1 },
	{ id: 'single-balcony', key: 'singleBalcony', total: 1 },
	{ id: 'single', key: 'single', total: 2 },
	{ id: 'double-twin', key: 'doubleTwin', total: 4 },
	{ id: 'double-euro', key: 'doubleEuro', total: 4 },
] as const

/**
 * Availability by room id for the given stay, or `null` when it is not known.
 * The backend only knows today's counts, so any check-in other than today is `null`. Errors are left to the caller.
 */
export async function getRoomAvailability(query: StayQuery): Promise<Record<string, RoomAvailability> | null> {
	if (query.checkIn !== todayInMoscow()) return null
	const hotel = await fetchHotelRooms()
	if (!hotel) return null

	const availability: Record<string, RoomAvailability> = {}
	for (const { id, key, total } of roomTypes) {
		const count = hotel.rooms[key]
		availability[id] =
			count === 0
				? { state: 'unavailable' }
				: total === 1
					? { state: 'available' }
					: { state: 'available', left: count }
	}
	return availability
}

/** Rooms free today, from the backend. `null` when it has no data yet; errors are left to the caller. */
export async function getFreeRoomsCount(): Promise<number | null> {
	return (await fetchHotelRooms())?.free ?? null
}
