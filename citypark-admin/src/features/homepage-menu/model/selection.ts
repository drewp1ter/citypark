import type { Product } from '../api'

export function moveItem<T>(items: T[], index: number, direction: -1 | 1): T[] {
	const target = index + direction
	if (index < 0 || index >= items.length || target < 0 || target >= items.length) return items
	const result = [...items]
	;[result[index], result[target]] = [result[target]!, result[index]!]
	return result
}

export function categoryProducts(products: Product[], categoryId: number, search: string): Product[] {
	const text = search.trim().toLocaleLowerCase('ru')
	return products.filter(
		(product) => product.categoryId === categoryId && product.name.toLocaleLowerCase('ru').includes(text)
	)
}
