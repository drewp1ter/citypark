import { PLACEHOLDER_PHOTO, typo } from '@/shared/lib'
import type { Dish, MenuCategory } from '../model/types'
import type { ApiCategory, ApiProduct } from './menu-api'

export function toCategory(c: ApiCategory): MenuCategory {
	return { id: String(c.id), name: c.name }
}

/** The API sends the weight in kilograms: 0.2 -> "200 г", 1.5 -> "1,5 кг". Empty when the weight is unknown. */
export function weightLabel(kg: number): string {
	if (kg <= 0) return ''
	if (kg < 1) return typo(`${Math.round(kg * 1000)} г`)
	return typo(`${String(Math.round(kg * 100) / 100).replace('.', ',')} кг`)
}

// next/image refuses hosts that next.config.ts does not list, so only the configured host is used.
function photoSrc(url: string | undefined): string {
	const host = process.env.MENU_IMAGE_HOST
	if (!url || !host) return PLACEHOLDER_PHOTO
	try {
		return new URL(url).host === host ? url : PLACEHOLDER_PHOTO
	} catch {
		return PLACEHOLDER_PHOTO
	}
}

export function toDish(p: ApiProduct): Dish {
	const weight = weightLabel(p.weight)
	return {
		id: String(p.id),
		externalId: String(p.id),
		name: p.name,
		categoryId: String(p.categoryId),
		composition: p.description.trim() || undefined,
		photo: { src: photoSrc(p.images[0]?.url), alt: p.name },
		variants: [{ id: 'default', label: weight, weight, price: p.price > 0 ? p.price : null }],
		available: p.isAvailableForOrder,
	}
}
