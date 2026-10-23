import { createFileRoute } from '@tanstack/react-router'
import { PageContent } from '../../../features/page-content/ui/page-content'

export const Route = createFileRoute('/_app/content/home')({
	component: () => <PageContent page='home' />,
})
