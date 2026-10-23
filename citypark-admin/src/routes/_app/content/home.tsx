import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../../shared/ui/page-header'

export const Route = createFileRoute('/_app/content/home')({
	component: () => (
		<>
			<PageHeader title='Главная' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
