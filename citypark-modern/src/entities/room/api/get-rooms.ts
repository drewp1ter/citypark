import { typo } from '@/shared/lib'
import { fetchHotelRooms } from './hotel-api'
import { fetchRooms } from './rooms-api'
import type { Room } from '../model/types'

// Room content comes from Go; full-stay availability is checked separately through the booking service.

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

/** Full-stay availability comes from the booking service, independently of today's counts. */
export { fetchStayAvailability as getRoomAvailability } from './booking-api'

/** Rooms free today, from the backend. `null` when it has no data yet; errors are left to the caller. */
export async function getFreeRoomsCount(): Promise<number | null> {
	return (await fetchHotelRooms())?.free ?? null
}
