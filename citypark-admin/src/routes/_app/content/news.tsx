import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../../shared/ui/page-header'

export const Route = createFileRoute('/_app/content/news')({
	component: () => (
		<>
			<PageHeader title='Новости' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
