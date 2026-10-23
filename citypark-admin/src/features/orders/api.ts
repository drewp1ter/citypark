import { keepPreviousData, queryOptions } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import type { paths } from '../../shared/api/schema'
import { PAGE_SIZE, toApiQuery, type OrderSearch } from './model/search'

type ListQuery = NonNullable<paths['/orders']['get']['parameters']['query']>

export function ordersQuery(search: OrderSearch) {
	return queryOptions({
		queryKey: ['orders', search],
		// toApiQuery builds only the documented parameters, with validated values.
		queryFn: () => unwrap(api.GET('/orders', { params: { query: toApiQuery(search, PAGE_SIZE) as ListQuery } })),
		placeholderData: keepPreviousData,
	})
}

export function orderQuery(id: number) {
	return queryOptions({
		queryKey: ['orders', 'details', id],
		queryFn: () => unwrap(api.GET('/orders/{id}', { params: { path: { id } } })),
		retry: false,
	})
}
