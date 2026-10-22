import Image from 'next/image'
import { site } from '@/shared/config'
import { Breadcrumbs, Button, StatTile } from '@/shared/ui'

/** Intro card, the façade photo and the facts. Shorter on mobile (two facts, no buttons). */
export function Hero() {
	return (
		<section className='max-w-page mx-auto w-full px-3 pt-4 md:px-6 md:pt-8'>
			<div className='hidden md:block'>
				<Breadcrumbs items={[{ label: 'Главная', href: '/' }, { label: 'Отель-люкс' }]} />
			</div>
			<div className='grid grid-cols-2 gap-2 md:grid-cols-4 md:gap-3'>
				<div className='border-border bg-surface col-span-2 flex flex-col gap-3 rounded-lg border p-6 md:justify-between md:gap-6 md:rounded-xl md:p-10'>
					<h1 className='heading text-[38px] md:text-[60px]'>
						Отель-
						<br className='max-md:hidden' />
						люкс
					</h1>
					<p className='text-text-muted md:text-lg'>
						<span className='md:hidden'>12 номеров у Городского парка.</span>
						<span className='max-md:hidden'>
							12 номеров в центре Белореченска. Рядом Городской парк, внизу — наш ресторан.
						</span>
					</p>
					<div className='flex flex-wrap gap-2 max-md:hidden'>
						<Button href='/rooms' size='lg'>
							Выбрать номер
						</Button>
						<Button href={site.phones.free.href} variant='outline' size='lg'>
							{site.phones.free.display}
						</Button>
					</div>
				</div>

				<div className='relative col-span-2 h-[240px] overflow-hidden rounded-lg md:h-[420px] md:rounded-xl'>
					<Image
						src='/images/hotel-building.jpg'
						alt='Фасад гостиницы'
						fill
						loading='eager'
						fetchPriority='high'
						sizes='(min-width: 1280px) 620px, (min-width: 768px) 50vw, 100vw'
						className='object-cover'
					/>
				</div>

				<ul className='contents'>
					<StatTile className='max-md:hidden' value='19–45 м²' caption='площадь номеров' />
					<StatTile
						value='24/7'
						caption={
							<>
								заезд<span className='max-md:hidden'>, выезд до 12:00</span>
							</>
						}
					/>
					<StatTile className='max-md:hidden' value='0 ₽' caption='Wi-Fi и парковка' />
					<StatTile
						accent
						value='0 ₽'
						caption={
							<>
								<span className='md:hidden'>бронь и Wi-Fi</span>
								<span className='max-md:hidden'>за бронирование</span>
							</>
						}
					/>
				</ul>
			</div>
		</section>
	)
}
