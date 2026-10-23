import { serverEnv } from '@/shared/config'

export type ApiRoom = {
	id: string
	kind: 'single' | 'double' | 'vip'
	balcony: boolean
	name: string
	hotelName: string
	numbers: string
	area: string
	areaDetailed: string
	capacity: string
	/** HTML. */
	description: string
	/** Plain text, one or two phrases for the card; may be empty. */
	summary: string
	pricePerNight: number
	/** `url` is a site path (`/images/…`) or a full URL of an uploaded image. */
	photo: { url: string; alt: string; position: string }
	hotelNote: string
	updatedAt: string
}

/** GET /rooms: the five room types in the order of the site. */
export async function fetchRooms(): Promise<ApiRoom[]> {
	const response = await fetch(serverEnv().MENU_API_URL + '/rooms', { next: { revalidate: 60 } })
	if (!response.ok) throw new Error(`GET /rooms: ${response.status}`)
	return (await response.json()) as ApiRoom[]
}
