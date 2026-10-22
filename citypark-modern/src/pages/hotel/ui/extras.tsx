import Image from 'next/image'
import { formatPrice } from '@/shared/lib'
import { cardClass } from '@/shared/ui'
import { breakfast, extras } from '../model/content'

/** The park photo, balconies, free extras and the breakfast menu. On mobile only the breakfast card is shown. */
export function Extras() {
	return (
		<section className='max-w-page mx-auto grid w-full gap-3 px-3 pt-11 md:grid-cols-2 md:px-6 md:pt-[104px] lg:grid-cols-3'>
			<div className='relative hidden min-h-[320px] overflow-hidden rounded-xl md:block'>
				<Image
					src='/images/hotel-park.jpg'
					alt='Гостиница среди деревьев'
					fill
					sizes='(min-width: 1024px) 400px, 50vw'
					className='object-cover'
				/>
			</div>

			<div className={cardClass('surface', 'hidden gap-2.5 rounded-xl p-8 md:flex')}>
				<h3 className='font-display text-2xl font-medium'>{extras.balconies.title}</h3>
				<p className='text-text-muted'>{extras.balconies.text}</p>
				<h3 className='font-display mt-3.5 text-2xl font-medium'>{extras.free.title}</h3>
				<p className='text-text-muted'>{extras.free.text}</p>
			</div>

			<div
				className={cardClass(
					'inverse',
					'rounded-lg p-[22px] md:col-span-2 md:gap-2.5 md:rounded-xl md:p-8 lg:col-span-1'
				)}
			>
				<h3 className='font-display mb-1 text-[22px] font-medium md:mb-0 md:text-2xl'>Завтрак в номер</h3>
				<p className='text-inverse-muted mb-2.5 text-sm md:mb-0 md:text-[15px]'>{breakfast.note}</p>
				<ul className='md:mt-1.5'>
					{breakfast.items.map((item) => (
						<li
							key={item.name}
							className={`border-inverse-border justify-between gap-3 border-b py-[9px] last:border-b-0 ${
								'desktopOnly' in item ? 'hidden md:flex' : 'flex'
							}`}
						>
							<span>
								{'shortName' in item ? (
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
