import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import type { components } from '../../shared/api/schema'
import { toastSuccess } from '../../shared/ui/toast'

export type HomepageMenu = components['schemas']['HomepageMenu']
export type HomepageCategory = HomepageMenu['categories'][number]
export type Category = components['schemas']['Category']
export type Product = components['schemas']['Product']

export const homepageMenuQuery = queryOptions({
	queryKey: ['homepage-menu', 'configuration'],
	queryFn: () => unwrap(api.GET('/homepage-menu')),
})
export const homepageCategoriesQuery = queryOptions({
	queryKey: ['homepage-menu', 'categories'],
	queryFn: () => unwrap(api.GET('/categories')),
})
export const homepageProductsQuery = queryOptions({
	queryKey: ['homepage-menu', 'products'],
	queryFn: async () => {
		const products: Product[] = []
		for (let page = 1; ; page++) {
			const batch = await unwrap(api.GET('/products', { params: { query: { page, pageSize: 500 } } }))
			products.push(...batch)
			if (batch.length < 500) return products
		}
	},
})

export function useSaveHomepageMenu() {
	const client = useQueryClient()
	return useMutation({
		mutationFn: (body: HomepageMenu) => unwrap(api.PUT('/homepage-menu', { body })),
		onSuccess: (data) => {
			client.setQueryData(homepageMenuQuery.queryKey, data)
			toastSuccess('Блюда на главной сохранены')
		},
	})
}
