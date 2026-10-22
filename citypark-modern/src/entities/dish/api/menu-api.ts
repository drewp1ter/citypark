import { serverEnv } from '@/shared/config'

export type ApiImage = { id: number; url: string; type: string }
export type ApiCategory = {
	id: number
	name: string
	description: string
	images: ApiImage[]
	isAvailableForOrder: boolean
}
export type ApiProduct = {
	id: number
	categoryId: number
	name: string
	description: string
	measureUnit: string
	/** Kilograms. */
	weight: number
	price: number
	images: ApiImage[]
	isAvailableForOrder: boolean
}

const PAGE_SIZE = 500

// 60 s: availability depends on the time of day.
async function fetchJson<T>(path: string): Promise<T> {
	const response = await fetch(serverEnv().MENU_API_URL + path, { next: { revalidate: 60 } })
	if (!response.ok) throw new Error(`GET ${path}: ${response.status}`)
	return (await response.json()) as T
}

export function fetchCategories(): Promise<ApiCategory[]> {
	return fetchJson<ApiCategory[]>('/categories')
}

export async function fetchProducts(): Promise<ApiProduct[]> {
	const products: ApiProduct[] = []
	for (let page = 1; ; page++) {
		const batch = await fetchJson<ApiProduct[]>(`/products?pageSize=${PAGE_SIZE}&page=${page}`)
		products.push(...batch)
		if (batch.length < PAGE_SIZE) return products
	}
}
