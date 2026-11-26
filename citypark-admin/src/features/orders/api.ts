import { keepPreviousData, queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import type { paths } from '../../shared/api/schema'
import { isSendingToIiko } from '../../shared/lib/iiko-status'
import { PAGE_SIZE, toApiQuery, type OrderSearch } from './model/search'

type ListQuery = NonNullable<paths['/orders']['get']['parameters']['query']>

export function ordersQuery(search: OrderSearch) {
	return queryOptions({
		queryKey: ['orders', search],
		// toApiQuery builds only the documented parameters, with validated values.
		queryFn: () => unwrap(api.GET('/orders', { params: { query: toApiQuery(search, PAGE_SIZE) as ListQuery } })),
		placeholderData: keepPreviousData,
		refetchInterval: (query) => {
			if (query.state.data?.orders.some((order) => isSendingToIiko(order.iikoStatus))) return 2000
			return search.iikoStatus === 'awaiting_approval' ? 10000 : false
		},
	})
}

export function orderQuery(id: number) {
	return queryOptions({
		queryKey: ['orders', 'details', id],
		queryFn: () => unwrap(api.GET('/orders/{id}', { params: { path: { id } } })),
		retry: false,
		refetchInterval: (query) => {
			const status = query.state.data?.iikoStatus
			if (isSendingToIiko(status)) return 2000
			return status === 'awaiting_approval' ? 10000 : false
		},
	})
}

type Approval = paths['/orders/{id}/approve']['post']['requestBody']['content']['application/json']

export function useApproveOrder(id: number) {
	const cache = useQueryClient()
	return useMutation({
		mutationFn: (body: Approval) => unwrap(api.POST('/orders/{id}/approve', { params: { path: { id } }, body })),
		onSuccess: async (order) => {
			await cache.cancelQueries({ queryKey: orderQuery(id).queryKey })
			cache.setQueryData(orderQuery(id).queryKey, order)
			await Promise.all([
				cache.invalidateQueries({ queryKey: ['orders'] }),
				cache.invalidateQueries({ queryKey: ['live-orders'] }),
			])
		},
	})
}
