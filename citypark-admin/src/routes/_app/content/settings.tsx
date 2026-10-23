import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../../shared/ui/page-header'

export const Route = createFileRoute('/_app/content/settings')({
	component: () => (
		<>
			<PageHeader title='Настройки сайта' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
