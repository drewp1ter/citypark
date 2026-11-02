import { createFileRoute } from '@tanstack/react-router'
import { PageContent } from '../../../features/page-content/ui/page-content'
import { HomepageMenuEditor } from '../../../features/homepage-menu/ui/homepage-menu-editor'

export const Route = createFileRoute('/_app/content/home')({
	component: () => (
		<>
			<PageContent page='home' />
			<HomepageMenuEditor />
		</>
	),
})
