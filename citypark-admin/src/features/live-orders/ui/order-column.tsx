import type { LiveOrder } from '../model/board'
import { LiveOrderCard } from './live-order-card'

export function OrderColumn({
	title,
	orders,
	freshIds,
	className = '',
}: {
	title: string
	orders: LiveOrder[]
	freshIds: ReadonlySet<number>
	className?: string
}) {
	return (
		<section aria-label={title} className={`flex min-w-0 flex-col gap-3 rounded-2xl bg-default/40 p-3 ${className}`}>
			<h2 className='font-semibold'>
				{title} · {orders.length}
			</h2>
			{orders.length === 0 ? (
				<p className='text-sm text-muted'>Пусто</p>
			) : (
				<ul className='flex flex-col gap-2'>
					{orders.map((order) => (
						<LiveOrderCard key={order.id} order={order} isFresh={freshIds.has(order.id)} />
					))}
				</ul>
			)}
		</section>
	)
}
