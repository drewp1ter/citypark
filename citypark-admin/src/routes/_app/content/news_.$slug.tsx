import { createFileRoute } from '@tanstack/react-router'
import { NewsEditPage } from '../../../features/news/ui/news-edit-page'

export const Route = createFileRoute('/_app/content/news_/$slug')({
	component: NewsRoute,
})

function NewsRoute() {
	const { slug } = Route.useParams()
	return <NewsEditPage slug={slug} />
}
