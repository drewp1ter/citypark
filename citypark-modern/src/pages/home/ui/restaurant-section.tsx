import Image from 'next/image'
import type { PageContent } from '@/entities/content'
import { TableBookingForm } from '@/features/book-table'
import { AppLink, Button, Card, cardClass, Section } from '@/shared/ui'

const promoTitle = 'font-display text-xl leading-[1.15] font-medium tracking-[-0.02em] md:text-2xl'

export function RestaurantSection({ content }: { content: PageContent }) {
	const terrace = content.image('home.restaurant.terrace-photo')

	return (
		<Section title={content.text('home.restaurant.heading')}>
			<div className='grid gap-2 md:grid-cols-2 md:gap-3 lg:grid-cols-3'>
				<div className='relative h-[220px] overflow-hidden rounded-lg md:col-span-2 md:h-[420px] md:rounded-xl'>
					{terrace && (
						<Image
							src={terrace.src}
							alt={terrace.alt}
							fill
							sizes='(min-width: 1280px) 820px, (min-width: 768px) 100vw, 100vw'
							style={{ objectPosition: terrace.position }}
							className='object-cover'
						/>
					)}
				</div>

				{/* On mobile the booking form is replaced by a button to the restaurant page. */}
				<TableBookingForm className='hidden md:col-span-2 md:flex lg:col-span-1 lg:row-span-2' />

				<Card
					variant='accent'
					className='gap-1.5 rounded-lg p-5 md:min-h-[220px] md:justify-between md:gap-4 md:rounded-xl md:p-7'
				>
					<span className={promoTitle}>{content.text('home.restaurant.banquet-title')}</span>
					<span className='text-[15px]'>
						<span className='md:hidden'>{content.text('home.restaurant.banquet-text-mobile')}</span>
						<span className='hidden md:inline'>{content.text('home.restaurant.banquet-text')}</span>
					</span>
				</Card>

				<AppLink
					href='/restaurant'
					className={cardClass('surface', 'hidden min-h-[220px] justify-between gap-4 rounded-xl p-7 md:flex')}
				>
					<span className={promoTitle}>{content.text('home.restaurant.panorama-title')}</span>
					<span className='text-text-muted text-[15px]'>{content.text('home.restaurant.panorama-text')}</span>
				</AppLink>

				<Button href='/restaurant' variant='inverse' size='xl' className='md:hidden'>
					Забронировать столик
				</Button>
			</div>
		</Section>
	)
}
