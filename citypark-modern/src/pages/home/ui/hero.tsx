import { Fragment } from 'react'
import Image from 'next/image'
import type { PageContent } from '@/entities/content'
import { delivery } from '@/entities/dish'
import { minPricePerNight, type Room } from '@/entities/room'
import { site } from '@/shared/config'
import { formatPrice, type Photo } from '@/shared/lib'
import { AppLink, Badge, StatTile } from '@/shared/ui'

type TileProps = {
	href: string
	photo: Photo | null
	/** Desktop label (title + caption) and the shorter mobile one. */
	title: { desktop: string; mobile: string }
	caption: { desktop: string; mobile: string }
	className?: string
	priority?: boolean
}

/** Photo tile with a white label at the bottom. The label is smaller on mobile and has an arrow circle on desktop. */
function PhotoTile({ href, photo, title, caption, className, priority }: TileProps) {
	return (
		<AppLink
			href={href}
			className={`relative block h-[200px] overflow-hidden rounded-md md:min-h-[300px] md:rounded-xl ${className ?? ''}`}
		>
			{photo && (
				<Image
					src={photo.src}
					alt={photo.alt}
					fill
					loading={priority ? 'eager' : undefined}
					fetchPriority={priority ? 'high' : undefined}
					sizes='(min-width: 1280px) 410px, (min-width: 768px) 50vw, 50vw'
					style={{ objectPosition: photo.position }}
					className='object-cover'
				/>
			)}
			<span className='bg-surface absolute inset-x-2 bottom-2 flex items-center justify-between gap-2 rounded-xs px-3 py-2.5 md:inset-x-3.5 md:bottom-3.5 md:rounded-sm md:px-4 md:py-3.5'>
				<span className='flex flex-col'>
					<strong className='text-[15px] md:text-base md:font-bold'>
						<span className='md:hidden'>{title.mobile}</span>
						<span className='hidden md:inline'>{title.desktop}</span>
					</strong>
					<span className='text-text-muted text-[13px] md:text-sm'>
						<span className='md:hidden'>{caption.mobile}</span>
						<span className='hidden md:inline'>{caption.desktop}</span>
					</span>
				</span>
				<span
					aria-hidden='true'
					className='bg-inverse-bg text-inverse-text hidden size-9 items-center justify-center rounded-full md:inline-flex'
				>
					↗
				</span>
			</span>
		</AppLink>
	)
}

const { phones, address } = site
const [phoneHead, phoneTail] = ['8 918', '311-97-91']

/** Two versions of one block: desktop `key` and `key-mobile`. */
function both(content: PageContent, key: string, values?: Readonly<Record<string, number>>) {
	return { desktop: content.text(key, values), mobile: content.text(`${key}-mobile`, values) }
}

/** The text of a block in both sizes, for places where only the desktop one differs in length. */
function Responsive({ text }: { text: { desktop: string; mobile: string } }) {
	return (
		<>
			<span className='md:hidden'>{text.mobile}</span>
			<span className='hidden md:inline'>{text.desktop}</span>
		</>
	)
}

/** Bento grid: intro card, three photo tiles, phone card, then the facts. */
export function Hero({ content, rooms }: { content: PageContent; rooms: readonly Room[] }) {
	const min = minPricePerNight(rooms)
	const price = min === null ? undefined : { minRoomPrice: min }
	const freeFrom = formatPrice(delivery.freeFrom)
	const deliveryCaption = both(content, 'home.hero.delivery-caption')

	return (
		<section className='max-w-page mx-auto w-full px-3 pt-4 md:px-6 md:pt-8'>
			<div className='grid grid-cols-2 gap-2 md:gap-3 lg:grid-cols-3'>
				<div className='border-border bg-surface col-span-2 flex flex-col justify-between gap-[18px] rounded-lg border p-6 md:min-h-[380px] md:gap-7 md:rounded-xl md:p-10'>
					<Badge dot='md'>
						<span className='hidden md:inline'>{address.city} · </span>
						{address.street}
					</Badge>
					<h1 className='heading text-[40px] md:text-[length:var(--text-hero)]'>
						{content.lines('home.hero.title').map((line, index) => (
							<Fragment key={line}>
								{index > 0 && <br />}
								{line}
							</Fragment>
						))}
					</h1>
					<p className='text-text-muted max-w-[520px] md:text-lg'>
						<Responsive text={both(content, 'home.hero.lead')} />
					</p>
				</div>

				<PhotoTile
					href='/rooms'
					photo={content.image('home.hero.hotel-photo')}
					title={both(content, 'home.hero.hotel-title')}
					caption={price ? both(content, 'home.hero.hotel-caption', price) : { desktop: '', mobile: '' }}
					className='lg:min-h-[380px]'
					priority
				/>
				<PhotoTile
					href='/restaurant'
					photo={content.image('home.hero.restaurant-photo')}
					title={both(content, 'home.hero.restaurant-title')}
					caption={both(content, 'home.hero.restaurant-caption')}
				/>
				<PhotoTile
					href='/menu'
					photo={content.image('home.hero.delivery-photo')}
					title={both(content, 'home.hero.delivery-title')}
					caption={{
						mobile: `${deliveryCaption.mobile} ${freeFrom}`,
						desktop: `${deliveryCaption.desktop} ${freeFrom}`,
					}}
				/>

				{/* Phone card: one big link on mobile, number + hotel number on desktop */}
				<AppLink
					href={phones.restaurant.href}
					className='bg-inverse-bg text-inverse-text flex h-[200px] flex-col justify-between rounded-md p-4 md:hidden'
				>
					<span className='text-inverse-muted text-[13px]'>{content.text('home.hero.phone-label-mobile')}</span>
					<span className='font-display text-[19px] leading-[1.2] font-medium'>
						{phoneHead}
						<br />
						{phoneTail}
					</span>
				</AppLink>
				<div className='bg-inverse-bg text-inverse-text hidden min-h-[300px] flex-col justify-between gap-[18px] rounded-xl p-8 md:flex'>
					<p className='text-inverse-muted text-sm'>{content.text('home.hero.phone-note')}</p>
					<AppLink
						href={phones.restaurant.href}
						className='font-display text-[28px] leading-[1.15] font-medium tracking-[-0.02em]'
					>
						{phoneHead}
						<br />
						{phoneTail}
					</AppLink>
					<p className='text-inverse-muted text-sm'>
						{content.text('home.hero.phone-hotel-label')}{' '}
						<AppLink href={phones.hotel.href} className='tap-area text-inverse-text'>
							{phones.hotel.display}
						</AppLink>
					</p>
				</div>
			</div>

			<ul className='mt-2 grid grid-cols-2 gap-2 md:mt-3 md:gap-3 lg:grid-cols-4'>
				<StatTile
					value={content.text('home.hero.stat-rooms-value')}
					caption={<Responsive text={both(content, 'home.hero.stat-rooms-caption')} />}
				/>
				<StatTile
					value={content.text('home.hero.stat-checkin-value')}
					caption={<Responsive text={both(content, 'home.hero.stat-checkin-caption')} />}
				/>
				<StatTile
					className='hidden md:block'
					value={content.text('home.hero.stat-wifi-value')}
					caption={content.text('home.hero.stat-wifi-caption')}
				/>
				<StatTile
					className='hidden md:block'
					value={content.text('home.hero.stat-cooking-value')}
					caption={content.text('home.hero.stat-cooking-caption')}
				/>
			</ul>
		</section>
	)
}
