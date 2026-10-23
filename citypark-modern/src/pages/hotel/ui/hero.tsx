import { Fragment } from 'react'
import Image from 'next/image'
import type { PageContent } from '@/entities/content'
import { site } from '@/shared/config'
import { Breadcrumbs, Button, StatTile } from '@/shared/ui'

/** Desktop `key` and mobile `key-mobile` of one block, for text that is shorter on mobile. */
function Responsive({ content, name }: { content: PageContent; name: string }) {
	return (
		<>
			<span className='md:hidden'>{content.text(`${name}-mobile`)}</span>
			<span className='max-md:hidden'>{content.text(name)}</span>
		</>
	)
}

/** Intro card, the façade photo and the facts. Shorter on mobile (two facts, no buttons). */
export function Hero({ content }: { content: PageContent }) {
	const photo = content.image('hotel.hero.photo')
	return (
		<section className='max-w-page mx-auto w-full px-3 pt-4 md:px-6 md:pt-8'>
			<div className='hidden md:block'>
				<Breadcrumbs items={[{ label: 'Главная', href: '/' }, { label: 'Отель-люкс' }]} />
			</div>
			<div className='grid grid-cols-2 gap-2 md:grid-cols-4 md:gap-3'>
				<div className='border-border bg-surface col-span-2 flex flex-col gap-3 rounded-lg border p-6 md:justify-between md:gap-6 md:rounded-xl md:p-10'>
					<h1 className='heading text-[38px] md:text-[60px]'>
						{content.lines('hotel.hero.title').map((line, index) => (
							<Fragment key={line}>
								{index > 0 && <br className='max-md:hidden' />}
								{line}
							</Fragment>
						))}
					</h1>
					<p className='text-text-muted md:text-lg'>
						<Responsive content={content} name='hotel.hero.lead' />
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
					{photo && (
						<Image
							src={photo.src}
							alt={photo.alt}
							fill
							loading='eager'
							fetchPriority='high'
							sizes='(min-width: 1280px) 620px, (min-width: 768px) 50vw, 100vw'
							className='object-cover'
							style={{ objectPosition: photo.position }}
						/>
					)}
				</div>

				<ul className='contents'>
					<StatTile
						className='max-md:hidden'
						value={content.text('hotel.hero.stat-area-value')}
						caption={content.text('hotel.hero.stat-area-caption')}
					/>
					<StatTile
						value={content.text('hotel.hero.stat-checkin-value')}
						caption={<Responsive content={content} name='hotel.hero.stat-checkin-caption' />}
					/>
					<StatTile
						className='max-md:hidden'
						value={content.text('hotel.hero.stat-wifi-value')}
						caption={content.text('hotel.hero.stat-wifi-caption')}
					/>
					<StatTile
						accent
						value={content.text('hotel.hero.stat-booking-value')}
						caption={<Responsive content={content} name='hotel.hero.stat-booking-caption' />}
					/>
				</ul>
			</div>
		</section>
	)
}
