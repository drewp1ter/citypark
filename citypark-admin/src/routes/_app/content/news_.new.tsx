import { createFileRoute } from '@tanstack/react-router'
import { NewsNewPage } from '../../../features/news/ui/news-new-page'

// `news_` keeps this page out of the list route's tree: it is not rendered inside the list.
export const Route = createFileRoute('/_app/content/news_/new')({
	component: NewsNewPage,
})
