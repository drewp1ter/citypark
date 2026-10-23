import { createFileRoute } from '@tanstack/react-router'
import { parseOrderSearch } from '../../features/orders/model/search'
import { OrdersPage } from '../../features/orders/ui/orders-page'

export const Route = createFileRoute('/_app/orders')({
	validateSearch: parseOrderSearch,
	component: OrdersPage,
})
