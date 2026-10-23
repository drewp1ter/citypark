import type { Photo } from '@/shared/lib'

export type RoomKind = 'single' | 'double' | 'vip'

export type Room = {
	id: string
	/** Name on the rooms page and in the booking flow. */
	name: string
	/** Name in the "Номерной фонд" list of the hotel page, when it differs. */
	hotelName?: string
	kind: RoomKind
	balcony: boolean
	/** "№ 204, 304" */
	numbers: string
	/** "23 + 7 м²" */
	area: string
	/** Longer wording used by the desktop rooms page ("45 м² + балкон 7 м²"). */
	areaDetailed?: string
	/** "до 4 гостей" */
	capacity?: string
	/** Sanitized HTML from the backend (`<p>…</p>`); `''` when there is none. */
	descriptionHtml: string
	/** Rubles per night. */
	pricePerNight: number
	photo: Photo
	/** Second line of the room in the hotel page list ("№ 201 · двуспальная кровать"). */
	hotelNote: string
}

/**
 * What the booking engine says about a room type for the chosen dates.
 * `left` is how many rooms of this type are free.
 */
export type RoomAvailability = { state: 'available'; left?: number } | { state: 'unavailable' }

export type StayQuery = {
	/** ISO dates, "2026-10-16". */
	checkIn: string
	checkOut: string
	guests: number
}
