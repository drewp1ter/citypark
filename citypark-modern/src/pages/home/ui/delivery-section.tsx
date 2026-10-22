import Image from 'next/image'
import { delivery, type Dish, type MenuCategory } from '@/entities/dish'
import { AddToCartButton } from '@/features/add-to-cart'
import { formatPrice } from '@/shared/lib'
import { AppLink, cardClass, Chip, Section } from '@/shared/ui'

type DeliverySectionProps = {
	/** Category tabs shown on desktop (the first one is selected). */
	categories: readonly MenuCategory[]
	/** The three dishes to show. */
	dishes: readonly Dish[]
}

/** Dish card: a row with a small photo on mobile, a card with a big photo from 768 px. */
function DishTeaser({ dish }: { dish: Dish }) {
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
						{variant.weight} · {variant.price !== null && formatPrice(variant.price)}
					</span>
				</span>
				<AddToCartButton dish={dish} />
			</div>
		</article>
	)
}

export function DeliverySection({ categories, dishes }: DeliverySectionProps) {
	return (
		<Section
			title='Доставка'
			action={
				<>
					<AppLink href='/menu' className='tap-area text-[15px] font-semibold md:hidden'>
						Меню ↗
					</AppLink>
					<div className='hidden flex-wrap gap-2 md:flex'>
						{categories.map((category, index) => (
							<Chip
								key={category.id}
								href={index === 0 ? '/menu' : `/menu?category=${category.id}`}
								selected={index === 0}
							>
								{category.name}
							</Chip>
						))}
					</div>
				</>
			}
		>
			<div className='flex flex-col gap-2 md:grid md:grid-cols-2 md:gap-3 lg:grid-cols-4'>
				{dishes.map((dish) => (
					<DishTeaser key={dish.id} dish={dish} />
				))}

				<AppLink
					href='/menu'
					className={cardClass('inverse', 'hidden min-h-[270px] justify-between gap-4 rounded-lg p-7 md:flex')}
				>
					<span className='font-display text-[22px] leading-[1.15] font-medium tracking-[-0.02em]'>
						Всё меню доставки
					</span>
					<span className='text-inverse-muted text-[15px]'>
						Бесплатно от {formatPrice(delivery.freeFrom)} · {delivery.hours} · {delivery.outskirtsFee}
					</span>
					<span
						aria-hidden='true'
						className='bg-accent text-on-accent inline-flex size-11 items-center justify-center self-start rounded-full text-xl'
					>
						↗
					</span>
				</AppLink>
			</div>
		</Section>
	)
}
