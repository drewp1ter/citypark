import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../shared/ui/page-header'

export const Route = createFileRoute('/_app/orders')({
	component: () => (
		<>
			<PageHeader title='Заказы' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
