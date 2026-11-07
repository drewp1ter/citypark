import { serverEnv } from '@/shared/config'
import type { DeliveryTown } from '../model/types'

type ApiDeliveryPrice = {
	regionCode?: 'RU-KDA' | 'RU-AD'
	id: number
	name: string
	value: string
	price: number
	freeFrom: number
}

/** Towns the restaurant delivers to, with flat prices. Cached for 60 s like the menu. */
export async function getDeliveryTowns(): Promise<DeliveryTown[]> {
	const response = await fetch(serverEnv().MENU_API_URL + '/delivery-prices', { next: { revalidate: 60 } })
	if (!response.ok) throw new Error(`GET /delivery-prices: ${response.status}`)
	const prices = (await response.json()) as ApiDeliveryPrice[]
	return prices.map(({ name, price, freeFrom, regionCode }) => ({
		name,
		regionCode: regionCode ?? 'RU-KDA',
		price,
		freeFrom: freeFrom ?? 0,
	}))
}
