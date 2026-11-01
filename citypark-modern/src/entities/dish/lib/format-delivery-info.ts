import { formatPrice, typo } from '@/shared/lib'
import { defaultDeliveryInfo, type DeliveryInfo } from '../model/delivery'
import type { DeliveryTown } from '../model/types'
import { matchDeliveryTown } from './match-delivery-town'
export function selectDeliveryCity(towns: readonly DeliveryTown[]): DeliveryTown | null {
	if (!Array.isArray(towns) || towns.some((town) => !town || typeof town.name !== 'string')) return null
	const town = matchDeliveryTown('Белореченск', towns)
	return town &&
		[town.price, town.freeFrom].every(
			(value) =>
				typeof value === 'number' &&
				Number.isFinite(value) &&
				value >= 0 &&
				Number.isSafeInteger(Math.round(value * 100))
		)
		? town
		: null
}
export function formatDeliveryInfo(value: unknown, towns: readonly DeliveryTown[]): DeliveryInfo {
	const city = selectDeliveryCity(towns)
	if (!value || typeof value !== 'object') return { ...defaultDeliveryInfo, city }
	const data = value as Record<string, unknown>
	const cooking = data.cooking,
		label = data.outskirtsLabel,
		min = data.outskirtsMin,
		max = data.outskirtsMax
	if (
		typeof cooking !== 'string' ||
		!cooking.trim() ||
		[...cooking.trim()].length > 100 ||
		typeof label !== 'string' ||
		!label.trim() ||
		[...label.trim()].length > 60 ||
		typeof min !== 'number' ||
		typeof max !== 'number' ||
		!Number.isInteger(min) ||
		!Number.isInteger(max) ||
		min < 0 ||
		max > 100000 ||
		min > max
	)
		return { ...defaultDeliveryInfo, city }
	const range = min === max ? formatPrice(min) : typo(`${formatPrice(min).replace(/\s*₽$/, '')}–${formatPrice(max)}`)
	return { cooking: typo(cooking), outskirtsFee: typo(`${label} ${range}`), outskirtsRange: range, city }
}
