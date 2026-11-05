'use client'

import { formatAddress } from '@/entities/address'
import { Button } from '@/shared/ui'
import type { AddressEntry } from '../model/types'
export function AddressCard({
	entry,
	unavailable,
	pending,
	onEdit,
	onDelete,
	onDefault,
}: {
	entry: AddressEntry
	unavailable: boolean
	pending: boolean
	onEdit: () => void
	onDelete: () => void
	onDefault: () => void
}) {
	const address = 'address' in entry ? entry.address : formatAddress(entry)
	return (
		<li className='border-border bg-surface flex min-w-0 flex-col gap-3 rounded-lg border p-5 [overflow-wrap:anywhere]'>
			<h2 className='text-lg font-semibold'>{entry.name || `${entry.city}, ${address}`}</h2>
			{entry.name && (
				<p>
					{entry.city}, {address}
				</p>
			)}
			{entry.isDefault && <p className='font-semibold'>Основной адрес</p>}
			{'needsReview' in entry && entry.needsReview && <p>Проверьте и уточните адрес</p>}
			{unavailable && <p>Доставка в этот населённый пункт сейчас недоступна</p>}
			<div className='flex flex-wrap gap-2'>
				<Button variant='outline' disabled={pending} onClick={onEdit}>
					Изменить
				</Button>
				<Button variant='outline' disabled={pending} onClick={onDelete}>
					Удалить
				</Button>
				{!entry.isDefault && (
					<Button variant='ghost' disabled={pending} onClick={onDefault}>
						Сделать основным
					</Button>
				)}
			</div>
		</li>
	)
}
