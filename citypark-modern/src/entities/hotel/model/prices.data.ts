import { typo } from '@/shared/lib'
import type { HotelPrices } from './types'

// Copied from design/reference/*/hotel.html.

export const hotelPrices: HotelPrices = {
	breakfastDelivery: 100,
	lateCheckoutPerHour: 500,
	smokingFine: 2000,
	breakfast: [
		{ name: typo('Омлет, 130 г'), shortName: 'Омлет', price: 250, desktopOnly: false },
		{ name: 'Сырники', shortName: '', price: 300, desktopOnly: false },
		{ name: 'Блины с творогом', shortName: '', price: 200, desktopOnly: true },
		{ name: 'Круассан с сёмгой', shortName: '', price: 350, desktopOnly: false },
		{ name: 'Каша с фруктами', shortName: '', price: 80, desktopOnly: true },
	],
}
