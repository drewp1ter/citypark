import Image from 'next/image'
import { cardClass } from '@/shared/ui'
import { venues } from '../model/content'

function VenueCard({ venue }: { venue: (typeof venues)[keyof typeof venues] }) {
	return (
		<div className={cardClass('surface', 'gap-2.5 rounded-lg p-[22px] md:gap-3.5 md:rounded-xl md:p-10')}>
			<h2 className='heading text-[26px] tracking-[-0.02em] md:text-[40px] md:tracking-[-0.03em]'>{venue.title}</h2>
			<p className='md:text-lg'>
				<span className='md:hidden'>{venue.short}</span>
				<span className='max-md:hidden'>{venue.lead}</span>
			</p>
			<p className='text-text-muted max-md:hidden'>{venue.text}</p>
		</div>
	)
}

function Photo({ src, alt }: { src: string; alt: string }) {
	return (
		<div className='relative h-[300px] overflow-hidden rounded-xl max-md:hidden'>
			<Image src={src} alt={alt} fill sizes='(min-width: 1280px) 610px, 50vw' className='object-cover' />
		</div>
	)
}

/** "Ресторан" and "Летнее кафе": a text card each; the photos are desktop-only. */
export function Venues() {
	return (
		<section
			id='tour'
			className='max-w-page mx-auto grid w-full gap-2 px-3 pt-12 md:grid-cols-2 md:gap-3 md:px-6 md:pt-[104px]'
		>
			<VenueCard venue={venues.restaurant} />
			<VenueCard venue={venues.cafe} />
			<Photo src='/images/summer-cafe-terrace.jpg' alt='Летнее кафе под зонтами' />
			<Photo src='/images/cafe-booth.jpg' alt='Кабинка в кафе' />
		</section>
	)
}
