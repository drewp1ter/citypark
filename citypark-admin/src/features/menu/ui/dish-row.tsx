import { Chip } from '@heroui/react'
import { formatRub } from '../../../shared/lib/format'
import type { ApiProduct } from '../model/group'
import { formatWeight } from '../model/weight'

export function DishRow({ dish }: { dish: ApiProduct }) {
	const photo = dish.images[0]?.url
	const weight = formatWeight(dish.weight, dish.measureUnit)
	return (
		<li className='flex items-center gap-3 rounded-xl border border-border bg-surface p-3'>
			{photo !== undefined ? (
				<img src={photo} alt='' loading='lazy' className='size-14 shrink-0 rounded-lg object-cover' />
			) : (
				<div
					role='img'
					aria-label='Нет фото'
					className='flex size-14 shrink-0 items-center justify-center rounded-lg bg-default text-center text-[10px] leading-tight text-muted'
				>
					Нет фото
				</div>
			)}
			<div className='min-w-0 flex-1'>
				<div className='font-medium'>{dish.name}</div>
				<div className='mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted'>
					{weight !== '' && <span>{weight}</span>}
					{!dish.isAvailableForOrder && (
						<Chip size='sm' color='warning' variant='soft'>
							Не в продаже сейчас
						</Chip>
					)}
				</div>
			</div>
			<div className='shrink-0 font-medium'>{formatRub(dish.price)}</div>
		</li>
	)
}
