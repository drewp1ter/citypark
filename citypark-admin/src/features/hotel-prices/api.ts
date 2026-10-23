import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toRequest, type HotelPricesForm } from './model/form'

const key = ['hotel-prices'] as const

export const hotelPricesQuery = queryOptions({
	queryKey: key,
	queryFn: () => unwrap(api.GET('/hotel/prices')),
})

export function useUpdateHotelPrices() {
	const queryClient = useQueryClient()
	return useMutation({
		mutationFn: (form: HotelPricesForm) => unwrap(api.PUT('/hotel/prices', { body: toRequest(form) })),
		onSuccess: (saved) => queryClient.setQueryData(key, saved),
	})
}
