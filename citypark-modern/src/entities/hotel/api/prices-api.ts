import { serverEnv } from '@/shared/config'
import type { HotelPrices } from '../model/types'

export type ApiHotelPrices = HotelPrices & { updatedAt: string }

/** GET /hotel/prices. */
export async function fetchHotelPrices(): Promise<ApiHotelPrices> {
	const response = await fetch(serverEnv().MENU_API_URL + '/hotel/prices', { next: { revalidate: 60 } })
	if (!response.ok) throw new Error(`GET /hotel/prices: ${response.status}`)
	return (await response.json()) as ApiHotelPrices
}
