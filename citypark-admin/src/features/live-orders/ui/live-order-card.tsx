import { Chip } from '@heroui/react'
import { formatRub } from '../../../shared/lib/format'
import { iikoStatusLabel } from '../../../shared/lib/iiko-status'
import { StatusChip } from '../../../shared/ui/order-chips'
import type { LiveOrder } from '../model/board'

const timeFormat = new Intl.DateTimeFormat('ru-RU', {
	timeZone: 'Europe/Moscow',
	hour: '2-digit',
	minute: '2-digit',
	hour12: false,
})

export function LiveOrderCard({ order, isFresh, onOpen }: { order: LiveOrder; isFresh: boolean; onOpen: () => void }) {
	const iikoLabel = iikoStatusLabel(order.iikoStatus)
	return (
		<li
			className={`relative flex flex-col gap-1 rounded-xl border bg-surface p-3 ${
				isFresh ? 'border-accent ring-2 ring-accent' : 'border-border'
			}`}
		>
			<div className='flex flex-wrap items-center gap-x-2 gap-y-1'>
				<button
					type='button'
					onClick={onOpen}
					aria-haspopup='dialog'
					className='cursor-pointer font-medium text-accent after:absolute after:inset-0 hover:underline'
					aria-label={`Заказ ${order.id}`}
				>
					№{order.id}
				</button>
				{iikoLabel === undefined ? (
					<StatusChip status={order.status} />
				) : (
					<Chip variant='soft' size='sm'>
						{iikoLabel}
					</Chip>
				)}
				<span className='ml-auto text-sm text-muted'>{timeFormat.format(new Date(order.createdAt))}</span>
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
