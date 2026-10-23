import { typo } from '@/shared/lib'
import { hotelPrices } from '../model/prices.data'
import type { HotelPrices } from '../model/types'
import { fetchHotelPrices } from './prices-api'

/** Prices from the backend; the static copy when it does not answer. */
export async function getHotelPrices(): Promise<HotelPrices> {
	try {
		const prices = await fetchHotelPrices()
		return {
			breakfastDelivery: prices.breakfastDelivery,
			lateCheckoutPerHour: prices.lateCheckoutPerHour,
			smokingFine: prices.smokingFine,
			breakfast: prices.breakfast.map((item) => ({
				name: typo(item.name),
				shortName: typo(item.shortName),
				price: item.price,
				desktopOnly: item.desktopOnly,
			})),
		}
	} catch (error) {
		console.error('hotel prices: using static data:', error)
		return hotelPrices
	}
}
