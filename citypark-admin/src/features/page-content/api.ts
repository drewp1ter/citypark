import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../shared/api'
import { toRequest, type BlockDraft } from './model/draft'
import type { ApiBlock } from './model/sections'

export type ContentPage = 'home' | 'hotel'

export function pageContentQuery(page: ContentPage) {
	return queryOptions({
		queryKey: ['page-content', page],
		queryFn: () => unwrap(api.GET('/content/blocks', { params: { query: { page } } })),
	})
}

export function useSaveBlock(page: ContentPage) {
	const queryClient = useQueryClient()
	return useMutation({
		mutationFn: ({ block, draft }: { block: ApiBlock; draft: BlockDraft }) =>
			unwrap(
				api.PUT('/content/blocks/{key}', {
					params: { path: { key: block.key } },
					body: toRequest(block, draft),
				})
			),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ['page-content', page] }),
	})
}
