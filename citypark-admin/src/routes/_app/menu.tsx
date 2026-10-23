import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../shared/ui/page-header'

export const Route = createFileRoute('/_app/menu')({
	component: () => (
		<>
			<PageHeader title='Меню' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
