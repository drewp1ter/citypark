import type { DeliveryTown } from '../model/types'

const prefix =
	/^(?:(?:г|город)|(?:п|пос|посёлок|поселок)|(?:х|хутор)|(?:ст|ст-ца|станица)|(?:с|село)|(?:д|деревня)|аул)\.?\s+/iu

export function normalizeLocality(value: string): string {
	return value.trim().replace(prefix, '').replace(/ё/giu, 'е').replace(/\s+/gu, ' ').toLocaleLowerCase('ru-RU')
}

export function matchDeliveryTown(
	value: string | null | undefined,
	towns: readonly DeliveryTown[]
): DeliveryTown | undefined {
	if (!value) return undefined
	const normalized = normalizeLocality(value)
	const matches = towns.filter((town) => normalizeLocality(town.name) === normalized)
	return matches.length === 1 ? matches[0] : undefined
}
