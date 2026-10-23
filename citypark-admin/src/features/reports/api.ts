import { queryOptions } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import type { Period } from './model/periods'

const FIVE_MINUTES = 5 * 60_000

// 1C can take up to 30 s: no retries (the page has its own «Повторить») and no refetch on focus.
const slowSource = { staleTime: FIVE_MINUTES, refetchOnWindowFocus: false, retry: false } as const

export function hotelDailyQuery(period: Period) {
	return queryOptions({
		queryKey: ['reports', 'hotel-daily', period.from, period.to] as const,
		queryFn: () => unwrap(api.GET('/reports/hotel/daily', { params: { query: period } })),
		...slowSource,
	})
}

export function hotelSalesQuery(period: Period) {
	return queryOptions({
		queryKey: ['reports', 'hotel-sales', period.from, period.to] as const,
		queryFn: () => unwrap(api.GET('/reports/hotel/sales', { params: { query: period } })),
		...slowSource,
	})
}
