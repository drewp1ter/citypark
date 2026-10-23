import { queryOptions } from '@tanstack/react-query'
import { api, ApiError, unwrap } from '../../shared/api'

/** Free rooms as last reported by the hotel's sender; null while it has not posted yet (404). */
export const freeRoomsQuery = queryOptions({
	queryKey: ['free-rooms'] as const,
	queryFn: async () => {
		try {
			return await unwrap(api.GET('/hotel/rooms'))
		} catch (e) {
			if (e instanceof ApiError && e.status === 404) return null
			throw e
		}
	},
	refetchInterval: 60_000,
})
