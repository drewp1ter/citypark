import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'

const PRODUCTS_PAGE_SIZE = 500

export const categoriesQuery = queryOptions({
	queryKey: ['menu', 'categories'] as const,
	queryFn: () => unwrap(api.GET('/categories')),
})

/** Every product: pages of 500 are loaded until one is shorter. */
export const productsQuery = queryOptions({
	queryKey: ['menu', 'products'] as const,
	queryFn: async () => {
		const all = []
		for (let page = 1; ; page++) {
			const chunk = await unwrap(api.GET('/products', { params: { query: { page, pageSize: PRODUCTS_PAGE_SIZE } } }))
			all.push(...chunk)
			if (chunk.length < PRODUCTS_PAGE_SIZE) return all
		}
	},
})

export function useSyncMenu() {
	const queryClient = useQueryClient()
	return useMutation({
		mutationFn: () => unwrap(api.POST('/iiko/sync')),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ['menu'] }),
	})
}
