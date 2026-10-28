import { createFileRoute } from '@tanstack/react-router'
import { LiveOrdersPage } from '../../features/live-orders/ui/live-orders-page'

export const Route = createFileRoute('/_app/orders_/live')({
	component: LiveOrdersPage,
})
