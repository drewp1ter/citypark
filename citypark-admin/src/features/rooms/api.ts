import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import type { components } from '../../shared/api/schema'
import { toRequest, type RoomForm } from './model/form'

export type RoomId = components['schemas']['Room']['id']

const listKey = ['rooms'] as const

export const roomsQuery = queryOptions({
	queryKey: listKey,
	queryFn: () => unwrap(api.GET('/rooms')),
})

export function useUpdateRoom() {
	const queryClient = useQueryClient()
	return useMutation({
		mutationFn: ({ id, form }: { id: RoomId; form: RoomForm }) =>
			unwrap(
				api.PUT('/rooms/{id}', {
					params: { path: { id } },
					body: toRequest(form),
				})
			),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: listKey }),
	})
}
