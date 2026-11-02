import Image from 'next/image'
import type { Dish } from '@/entities/dish'
import { AddToCartButton } from '@/features/add-to-cart'
import { formatPrice } from '@/shared/lib'
import { cardClass } from '@/shared/ui'

/** A row on mobile and a photo card from 768 px. */
export function DishTeaser({ dish }: { dish: Dish }) {
	const variant = dish.variants[0]
	return (
		<article
			className={cardClass(
				'surface',
				'flex-row items-center gap-3 rounded-sm p-2 md:flex-col md:items-stretch md:gap-0 md:rounded-lg md:p-2.5'
			)}
		>
			<div className='relative size-20 shrink-0 overflow-hidden rounded-xs md:h-[200px] md:w-full md:rounded-sm'>
				<Image
					src={dish.photo.src}
					alt={dish.photo.alt}
					fill
					sizes='(min-width: 1024px) 25vw, (min-width: 768px) 50vw, 80px'
					className='object-cover'
				/>
			</div>
			<div className='flex flex-1 items-center justify-between gap-2 md:px-2.5 md:pt-3.5 md:pb-1.5'>
				<span className='flex flex-col'>
					<strong>{dish.name}</strong>
					<span className='text-text-muted text-sm'>
						{variant.weight}
						{variant.weight && variant.price !== null ? ' · ' : ''}
						{variant.price !== null && formatPrice(variant.price)}
					</span>
				</span>
				<AddToCartButton dish={dish} />
			</div>
		</article>
	)
}
