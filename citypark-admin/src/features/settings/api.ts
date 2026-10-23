import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toRequest, type SettingsForm } from './model/form'

const key = ['settings'] as const

export const settingsQuery = queryOptions({
	queryKey: key,
	queryFn: () => unwrap(api.GET('/settings')),
})

export function useUpdateSettings() {
	const queryClient = useQueryClient()
	return useMutation({
		mutationFn: (form: SettingsForm) => unwrap(api.PUT('/settings', { body: toRequest(form) })),
		onSuccess: (saved) => queryClient.setQueryData(key, saved),
	})
}
