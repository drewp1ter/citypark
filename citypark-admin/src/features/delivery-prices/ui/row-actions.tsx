import { Button } from '@heroui/react'
import type { DeliveryPrice } from './delivery-prices-page'

function Icon({ children }: { children: React.ReactNode }) {
	return (
		<svg
			width='18'
			height='18'
			viewBox='0 0 24 24'
			fill='none'
			stroke='currentColor'
			strokeWidth='2'
			strokeLinecap='round'
			strokeLinejoin='round'
			aria-hidden='true'
		>
			{children}
		</svg>
	)
}

export function RowActions({
	item,
	onEdit,
	onDelete,
}: {
	item: DeliveryPrice
	onEdit: (item: DeliveryPrice) => void
	onDelete: (item: DeliveryPrice) => void
}) {
	return (
		<div className='flex gap-1'>
			<Button isIconOnly size='sm' variant='ghost' aria-label={`Изменить ${item.name}`} onPress={() => onEdit(item)}>
				<Icon>
					<path d='M12 20h9' />
					<path d='M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z' />
				</Icon>
			</Button>
			<Button isIconOnly size='sm' variant='ghost' aria-label={`Удалить ${item.name}`} onPress={() => onDelete(item)}>
				<Icon>
					<path d='M3 6h18' />
					<path d='M8 6V4h8v2' />
					<path d='M19 6l-1 14H6L5 6' />
				</Icon>
			</Button>
		</div>
	)
}
