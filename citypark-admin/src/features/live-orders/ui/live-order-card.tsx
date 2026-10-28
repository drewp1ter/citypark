import { Link } from '@tanstack/react-router'
import { formatRub } from '../../../shared/lib/format'
import type { LiveOrder } from '../model/board'

const timeFormat = new Intl.DateTimeFormat('ru-RU', {
	timeZone: 'Europe/Moscow',
	hour: '2-digit',
	minute: '2-digit',
	hour12: false,
})

export function LiveOrderCard({ order, isFresh }: { order: LiveOrder; isFresh: boolean }) {
	return (
		<li
			className={`relative flex flex-col gap-1 rounded-xl border bg-surface p-3 ${
				isFresh ? 'border-accent ring-2 ring-accent' : 'border-border'
			}`}
		>
			<div className='flex items-center justify-between gap-2'>
				<Link
					to='/orders/$orderId'
					params={{ orderId: String(order.id) }}
					className='font-medium text-accent after:absolute after:inset-0 hover:underline'
					aria-label={`Заказ ${order.id}`}
				>
					№{order.id}
				</Link>
				<span className='text-sm text-muted'>{timeFormat.format(new Date(order.createdAt))}</span>
			</div>
			{order.deliveryTime !== '' && <div className='text-sm'>Ко времени {order.deliveryTime}</div>}
			<div>{order.customer.name}</div>
			<a href={`tel:${order.customer.phone}`} className='relative z-10 w-fit text-sm text-muted hover:underline'>
				{order.customer.phone}
			</a>
			<div className='text-sm'>
				{order.address === null ? 'Самовывоз' : `${order.address.city}, ${order.address.address}`}
			</div>
			{order.comment !== '' && <p className='line-clamp-2 text-sm text-muted'>{order.comment}</p>}
			<span className='mt-1 font-medium'>{formatRub(order.total + order.deliveryPrice)}</span>
		</li>
	)
}
