import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../shared/ui/page-header'

export const Route = createFileRoute('/_app/reports')({
	component: () => (
		<>
			<PageHeader title='Отчёты' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
