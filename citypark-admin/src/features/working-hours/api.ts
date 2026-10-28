import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toRequest, type WorkingHoursForm } from './model/form'

const key = ['working-hours'] as const

export const workingHoursQuery = queryOptions({
	queryKey: key,
	queryFn: () => unwrap(api.GET('/working-hours')),
})

export function useSaveWorkingHours() {
	const queryClient = useQueryClient()
	return useMutation({
		mutationFn: (form: WorkingHoursForm) => unwrap(api.PUT('/working-hours', { body: toRequest(form) })),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: key }),
	})
}
