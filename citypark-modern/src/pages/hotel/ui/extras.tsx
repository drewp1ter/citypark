import Image from 'next/image'
import type { PageContent } from '@/entities/content'
import type { HotelPrices } from '@/entities/hotel'
import { formatPrice } from '@/shared/lib'
import { cardClass, RichText } from '@/shared/ui'

/** The park photo, balconies, free extras and the breakfast menu. On mobile only the breakfast card is shown. */
export function Extras({ content, prices }: { content: PageContent; prices: HotelPrices }) {
	const photo = content.image('hotel.extras.photo')
	const values = { breakfastDelivery: prices.breakfastDelivery }

	return (
		<section className='max-w-page mx-auto grid w-full gap-3 px-3 pt-11 md:grid-cols-2 md:px-6 md:pt-[104px] lg:grid-cols-3'>
			<div className='relative hidden min-h-[320px] overflow-hidden rounded-xl md:block'>
				{photo && (
					<Image
						src={photo.src}
						alt={photo.alt}
						fill
						sizes='(min-width: 1024px) 400px, 50vw'
						className='object-cover'
						style={{ objectPosition: photo.position }}
					/>
				)}
			</div>

			<div className={cardClass('surface', 'hidden gap-2.5 rounded-xl p-8 md:flex')}>
				<h3 className='font-display text-2xl font-medium'>{content.text('hotel.extras.balconies-title')}</h3>
				<RichText html={content.html('hotel.extras.balconies-text')} variant='short' className='text-text-muted' />
				<h3 className='font-display mt-3.5 text-2xl font-medium'>{content.text('hotel.extras.free-title')}</h3>
				<RichText html={content.html('hotel.extras.free-text')} variant='short' className='text-text-muted' />
			</div>

			<div
				className={cardClass(
					'inverse',
					'rounded-lg p-[22px] md:col-span-2 md:gap-2.5 md:rounded-xl md:p-8 lg:col-span-1'
				)}
			>
				<h3 className='font-display mb-1 text-[22px] font-medium md:mb-0 md:text-2xl'>
					{content.text('hotel.extras.breakfast-heading')}
				</h3>
				<p className='text-inverse-muted mb-2.5 text-sm md:mb-0 md:text-[15px]'>
					{content.text('hotel.extras.breakfast-note', values)}
				</p>
				<ul className='md:mt-1.5'>
					{prices.breakfast.map((item) => (
						<li
							key={item.name}
							className={`border-inverse-border justify-between gap-3 border-b py-[9px] last:border-b-0 ${
								item.desktopOnly ? 'hidden md:flex' : 'flex'
							}`}
						>
							<span>
								{item.shortName ? (
									<>
										<span className='md:hidden'>{item.shortName}</span>
										<span className='max-md:hidden'>{item.name}</span>
									</>
								) : (
									item.name
								)}
							</span>
							<span className='text-accent'>{formatPrice(item.price)}</span>
						</li>
					))}
				</ul>
			</div>
		</section>
	)
}
