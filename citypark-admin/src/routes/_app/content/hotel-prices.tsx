import { createFileRoute } from '@tanstack/react-router'
import { PageHeader } from '../../../shared/ui/page-header'

export const Route = createFileRoute('/_app/content/hotel-prices')({
	component: () => (
		<>
			<PageHeader title='Цены отеля' />
			<p className='text-muted'>Раздел в разработке</p>
		</>
	),
})
