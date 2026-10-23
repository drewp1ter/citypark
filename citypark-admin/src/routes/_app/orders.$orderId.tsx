import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { OrderDetails } from '../../features/orders/ui/order-details'

export const Route = createFileRoute('/_app/orders/$orderId')({
	component: OrderRoute,
})

function OrderRoute() {
	const { orderId } = Route.useParams()
	const search = Route.useSearch()
	const navigate = useNavigate()
	return <OrderDetails orderId={orderId} onClose={() => void navigate({ to: '/orders', search })} />
}
