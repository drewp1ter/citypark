import { createFileRoute } from '@tanstack/react-router'
import { parsePage } from '../../../features/news/model/search'
import { NewsPage } from '../../../features/news/ui/news-page'

export const Route = createFileRoute('/_app/content/news')({
	validateSearch: (raw): { page?: number } => {
		const page = parsePage(raw.page)
		return page === undefined ? {} : { page }
	},
	component: NewsPage,
})
