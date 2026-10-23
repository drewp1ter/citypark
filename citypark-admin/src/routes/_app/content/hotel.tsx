import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../../shared/ui/page-header'

export const Route = createFileRoute('/_app/content/hotel')({
	component: () => (
		<>
			<PageHeader title='Отель' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
