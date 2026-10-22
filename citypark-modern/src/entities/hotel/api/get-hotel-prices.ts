import { hotelPrices } from '../model/prices.data'
import type { HotelPrices } from '../model/types'

export async function getHotelPrices(): Promise<HotelPrices> {
	// Swap the body for the API in task 20.
	return hotelPrices
}
