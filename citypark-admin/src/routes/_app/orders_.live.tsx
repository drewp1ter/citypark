import { createFileRoute } from '@tanstack/react-router'
import { LiveOrdersPage } from '../../features/live-orders/ui/live-orders-page'
import { useState } from 'react'
import { OrderDetails } from '../../features/orders/ui/order-details'

function LiveOrdersRoute() {
	const [orderId, setOrderId] = useState<number | null>(null)

	return (
		<>
			<LiveOrdersPage onOpenOrder={setOrderId} />
			{orderId !== null && (
				<OrderDetails orderId={String(orderId)} onClose={() => setOrderId(null)} closeLabel='Назад к заказам онлайн' />
			)}
		</>
	)
}

export const Route = createFileRoute('/_app/orders_/live')({
	component: LiveOrdersRoute,
})
