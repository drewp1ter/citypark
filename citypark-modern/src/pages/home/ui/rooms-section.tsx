import Image from 'next/image'
import type { PageContent } from '@/entities/content'
import { hasPriceRange, minPricePerNight, type Room } from '@/entities/room'
import { StayDatesBar } from '@/features/search-rooms'
import { formatPrice, typo } from '@/shared/lib'
import { AppLink, Button, Card, Section } from '@/shared/ui'
import { FreeRooms } from './free-rooms'

// The home page shows three short cards, with their own wording (not the same as the rooms page).
type Teaser = {
	roomIds: readonly string[]
	name: string
	details: { desktop: string; mobile: string }
	/** Key of the photo block. */
	photoKey: string
	featured?: boolean
}

const teasers: Teaser[] = [
	{
		roomIds: ['single', 'single-balcony'],
		name: 'Одноместный',
		details: { desktop: typo('19 м² · № 201 — с балконом'), mobile: typo('19 м²') },
		photoKey: 'home.rooms.single-photo',
	},
	{
		roomIds: ['double-twin', 'double-euro'],
		name: 'Двухместный',
		details: { desktop: typo('23–25 м² · 2 кровати или «евро»'), mobile: typo('23–25 м²') },
		photoKey: 'home.rooms.double-photo',
	},
	{
		roomIds: ['vip'],
		name: 'VIP-номер',
		details: { desktop: typo('45 м² · 2 комнаты · балкон · до 4 гостей'), mobile: typo('45 м² · до 4 гостей') },
		photoKey: 'home.rooms.vip-photo',
		featured: true,
	},
]

type RoomsSectionProps = { content: PageContent; rooms: readonly Room[] | null; free: number | null }

export function RoomsSection({ content, rooms, free }: RoomsSectionProps) {
	return (
		<Section
			title={content.text('home.rooms.heading')}
			action={
				<>
					<AppLink href='/rooms' className='tap-area text-[15px] font-semibold md:hidden'>
						Все ↗
					</AppLink>
					<StayDatesBar className='hidden md:flex' />
				</>
			}
		>
			{/* Mobile: horizontal rail that bleeds to the screen edge. From 768 px: grid. */}
			{rooms && (
				<div className='-mx-3 flex gap-2 overflow-x-auto px-3 pb-1 md:mx-0 md:grid md:grid-cols-2 md:gap-3 md:overflow-visible md:px-0 md:pb-0 lg:grid-cols-3'>
					{teasers.map((teaser) => {
						const photo = content.image(teaser.photoKey)

						return (
							<Card
								as='article'
								key={teaser.roomIds.join()}
								variant={teaser.featured ? 'inverse' : 'surface'}
								className='w-[270px] shrink-0 rounded-lg p-2 md:w-auto md:rounded-xl md:p-2.5'
							>
								{/* Mobile cards have no button: a link over the whole card instead. */}
								<AppLink
									href='/rooms'
									aria-label={`${teaser.name}: выбрать номер`}
									className='absolute inset-0 z-10 md:hidden'
								/>
								<div className='relative h-[170px] overflow-hidden rounded-sm md:h-[250px] md:rounded-md'>
									{photo && (
										<Image
											src={photo.src}
											alt={photo.alt}
											fill
											sizes='(min-width: 1024px) 400px, (min-width: 768px) 50vw, 270px'
											style={{ objectPosition: photo.position }}
											className='object-cover'
										/>
									)}
								</div>
								<div className='flex flex-1 flex-col gap-1.5 px-2 pt-3 pb-1.5 md:px-3.5 md:pt-[18px] md:pb-3'>
									<h3 className='text-lg font-bold md:text-[22px]'>{teaser.name}</h3>
									<p
										className={`flex-1 text-sm md:text-base ${teaser.featured ? 'text-inverse-muted' : 'text-text-muted'}`}
									>
										<span className='md:hidden'>{teaser.details.mobile}</span>
										<span className='hidden md:inline'>{teaser.details.desktop}</span>
									</p>
									<div className='mt-0.5 flex items-center justify-between gap-3 md:mt-3.5'>
										<span className='font-display text-lg font-medium md:text-[22px] md:tracking-[-0.02em]'>
											{hasPriceRange(rooms, teaser.roomIds) && 'от '}
											{formatPrice(minPricePerNight(rooms, teaser.roomIds) ?? 0)}
										</span>
										<Button
											href='/rooms'
											variant={teaser.featured ? 'accent' : 'inverse'}
											className='hidden px-5 md:inline-flex'
										>
											Выбрать
										</Button>
									</div>
								</div>
							</Card>
						)
					})}
				</div>
			)}
			{free !== null && <FreeRooms count={free} />}
		</Section>
	)
}
