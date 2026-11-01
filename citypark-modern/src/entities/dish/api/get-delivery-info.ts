import { serverEnv } from '@/shared/config'
import { formatDeliveryInfo } from '../lib/format-delivery-info'
import type { DeliveryInfo } from '../model/delivery'
import { getDeliveryTowns } from './get-delivery-prices'
export async function getDeliveryInfo(): Promise<DeliveryInfo> {
	const [settings, towns] = await Promise.allSettled([
		fetch(serverEnv().MENU_API_URL + '/delivery-settings', { next: { revalidate: 60 } }).then(async (response) => {
			if (!response.ok) throw new Error('Delivery settings unavailable')
			return response.json() as Promise<unknown>
		}),
		getDeliveryTowns(),
	])
	if (settings.status === 'rejected') console.error('Delivery settings unavailable')
	if (towns.status === 'rejected') console.error('Delivery towns unavailable')
	return formatDeliveryInfo(
		settings.status === 'fulfilled' ? settings.value : null,
		towns.status === 'fulfilled' ? towns.value : []
	)
}
