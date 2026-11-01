import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toRequest, type DeliverySettingsForm } from './model/form'
const key = ['delivery-settings'] as const
export const deliverySettingsQuery = queryOptions({
	queryKey: key,
	queryFn: () => unwrap(api.GET('/delivery-settings')),
})
export function useSaveDeliverySettings() {
	const client = useQueryClient()
	return useMutation({
		mutationFn: (form: DeliverySettingsForm) => unwrap(api.PUT('/delivery-settings', { body: toRequest(form) })),
		onSuccess: (saved) => {
			client.setQueryData(key, saved)
		},
	})
}
