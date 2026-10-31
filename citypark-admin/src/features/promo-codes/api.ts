import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toastSuccess } from '../../shared/ui/toast'
import type { CampaignInput } from './model/form'

export const campaignsQuery = queryOptions({
	queryKey: ['promo-codes', 'campaigns'],
	queryFn: () => unwrap(api.GET('/promo-codes')),
})
export const campaignQuery = (id: number) =>
	queryOptions({
		queryKey: ['promo-codes', 'campaigns', id],
		queryFn: () => unwrap(api.GET('/promo-codes/{id}', { params: { path: { id } } })),
	})
export const categoriesQuery = queryOptions({
	queryKey: ['promo-codes', 'categories'],
	queryFn: () => unwrap(api.GET('/categories')),
})
export const productsQuery = queryOptions({
	queryKey: ['promo-codes', 'products'],
	queryFn: async () => {
		const all = []
		for (let page = 1; ; page++) {
			const chunk = await unwrap(api.GET('/products', { params: { query: { page, pageSize: 500 } } }))
			all.push(...chunk)
			if (chunk.length < 500) return all
		}
	},
})
export function useSaveCampaign(id?: number) {
	const client = useQueryClient()
	return useMutation({
		mutationFn: (body: CampaignInput) =>
			id === undefined
				? unwrap(api.POST('/promo-codes', { body }))
				: unwrap(api.PUT('/promo-codes/{id}', { params: { path: { id } }, body })),
		onSuccess: () => {
			toastSuccess('Промокод сохранён')
			return client.invalidateQueries({ queryKey: ['promo-codes', 'campaigns'] })
		},
	})
}
