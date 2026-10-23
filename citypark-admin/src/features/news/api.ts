import { keepPreviousData, queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toRequest, type NewsForm } from './model/form'

export const NEWS_PAGE_SIZE = 20

const listKey = ['news'] as const

export function newsListQuery(page: number) {
	return queryOptions({
		queryKey: ['news', 'list', page],
		queryFn: () => unwrap(api.GET('/news', { params: { query: { page, pageSize: NEWS_PAGE_SIZE } } })),
		placeholderData: keepPreviousData,
	})
}

export function newsItemQuery(slug: string) {
	return queryOptions({
		queryKey: ['news', 'item', slug],
		queryFn: () => unwrap(api.GET('/news/{slug}', { params: { path: { slug } } })),
		retry: false,
	})
}

function useInvalidate() {
	const queryClient = useQueryClient()
	return () => queryClient.invalidateQueries({ queryKey: listKey })
}

export function useCreateNews() {
	const invalidate = useInvalidate()
	return useMutation({
		mutationFn: (form: NewsForm) => unwrap(api.POST('/news', { body: toRequest(form) })),
		onSuccess: invalidate,
	})
}

export function useUpdateNews() {
	const invalidate = useInvalidate()
	return useMutation({
		mutationFn: ({ id, form }: { id: number; form: NewsForm }) =>
			unwrap(api.PUT('/news/{id}', { params: { path: { id } }, body: toRequest(form) })),
		onSuccess: invalidate,
	})
}

export function useDeleteNews() {
	const invalidate = useInvalidate()
	return useMutation({
		mutationFn: (id: number) => unwrap(api.DELETE('/news/{id}', { params: { path: { id } } })),
		onSuccess: invalidate,
	})
}
