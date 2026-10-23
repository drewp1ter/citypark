import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toRequest, type DeliveryPriceForm } from './model/form'

const listKey = ['delivery-prices'] as const

export const deliveryPricesQuery = queryOptions({
	queryKey: listKey,
	queryFn: () => unwrap(api.GET('/delivery-prices')),
})

function useInvalidate() {
	const queryClient = useQueryClient()
	return () => queryClient.invalidateQueries({ queryKey: listKey })
}

export function useCreateDeliveryPrice() {
	const invalidate = useInvalidate()
	return useMutation({
		mutationFn: (form: DeliveryPriceForm) =>
			unwrap(api.POST('/delivery-prices', { body: { deliveryPrices: [toRequest(form)] } })),
		onSuccess: invalidate,
	})
}

export function useUpdateDeliveryPrice() {
	const invalidate = useInvalidate()
	return useMutation({
		mutationFn: ({ id, form }: { id: number; form: DeliveryPriceForm }) =>
			unwrap(api.PATCH('/delivery-prices/{id}', { params: { path: { id } }, body: toRequest(form) })),
		onSuccess: invalidate,
	})
}

export function useDeleteDeliveryPrice() {
	const invalidate = useInvalidate()
	return useMutation({
		mutationFn: (id: number) => unwrap(api.DELETE('/delivery-prices/{id}', { params: { path: { id } } })),
		onSuccess: invalidate,
	})
}
