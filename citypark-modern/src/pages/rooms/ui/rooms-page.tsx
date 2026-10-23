import { getPageContent } from '@/entities/content'
import { getHotelPrices } from '@/entities/hotel'
import { getRoomAvailability, getRooms, type Room } from '@/entities/room'
import { StayDatesBar } from '@/features/search-rooms'
import { site } from '@/shared/config'
import { cn, fillPlaceholders, plural } from '@/shared/lib'
import { AppLink, Breadcrumbs, Button, Chip, cardClass, Section } from '@/shared/ui'
import { roomsHref, type RoomsQuery, type RoomType } from '../model/query'
import { RoomCard } from './room-card'

const filters: { type: RoomType; label: string }[] = [
	{ type: 'single', label: 'Одноместные' },
	{ type: 'double', label: 'Двухместные' },
	{ type: 'balcony', label: 'С балконом' },
]

const matchesType = (room: Room, type: RoomType) => {
	if (type === 'single') return room.kind === 'single'
	if (type === 'double') return room.kind === 'double'
	if (type === 'balcony') return room.balcony
	return true
}

const chipLi = 'rounded-pill bg-bg px-3 py-1.5 text-sm md:px-4 md:py-2 md:text-base'

/** The rooms page: search bar, filters, the room cards, what every room has, and the phone numbers. */
export async function RoomsPage({ query }: { query: RoomsQuery }) {
	const [rooms, availability, content, prices] = await Promise.all([
		getRooms().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getRoomAvailability({
			checkIn: query.checkIn,
			checkOut: query.checkOut,
			guests: query.guests === '3-4' ? 3 : Number(query.guests),
		}).catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getPageContent('hotel'),
		getHotelPrices(),
	])

	if (!rooms) {
		return (
			<main className='max-w-page mx-auto flex w-full flex-col items-start gap-4 px-3 pt-4 pb-16 md:px-6 md:pt-8 md:pb-28'>
				<h1 className='heading mx-1.5 mt-2 text-[38px] md:mx-0 md:mt-0 md:text-[length:var(--text-h1)]'>Номера</h1>
				<p className='text-text-muted mx-1.5 md:mx-0 md:text-lg'>Не удалось загрузить номера. Позвоните нам:</p>
				<Button href={site.phones.hotel.href} size='xl' variant='outline'>
					{site.phones.hotel.display}
				</Button>
			</main>
		)
	}

	const amenities = content.lines('hotel.amenities')
	const mobileAmenities = content.lines('hotel.amenities-mobile')
	const quickFacts = content.lines('hotel.quick-facts')
	const values = { smokingFine: prices.smokingFine }

	const visible = rooms.filter(
		(room) =>
			matchesType(room, query.type) &&
			(!query.freeOnly || availability === null || availability[room.id]?.state === 'available')
	)

	return (
		<main>
			<section className='max-w-page mx-auto w-full px-3 pt-4 md:px-6 md:pt-8'>
				<div className='hidden md:block'>
					<Breadcrumbs
						items={[{ label: 'Главная', href: '/' }, { label: 'Отель', href: '/hotel' }, { label: 'Номера' }]}
					/>
				</div>
				<div className='flex flex-wrap items-end justify-between gap-5'>
					<div className='max-md:w-full'>
						<h1 className='heading mx-1.5 mt-2 mb-2 text-[38px] md:mx-0 md:mt-0 md:mb-3 md:text-[length:var(--text-h1)]'>
							Номера
						</h1>
						<p className='text-text-muted mx-1.5 mb-3.5 md:mx-0 md:mb-0 md:text-lg'>
							12 номеров<span className='hidden md:inline'> от 19 до 45&nbsp;м²</span> · без предоплаты
						</p>
					</div>
					<StayDatesBar
						variant='rooms'
						submitLabel='Найти'
						defaultCheckIn={query.given.checkIn}
						defaultCheckOut={query.given.checkOut}
						defaultGuests={query.guests}
						preserve={{
							...(query.type !== 'all' && { type: query.type }),
							...(query.freeOnly && { free: '1' }),
						}}
						className='max-md:w-full'
					/>
				</div>

				<div
					role='group'
					aria-label='Фильтр'
					className='-mx-3 mt-3 flex gap-1.5 overflow-x-auto px-3 md:mx-0 md:mt-6 md:flex-wrap md:gap-2 md:overflow-visible md:px-0'
				>
					<Chip href={roomsHref(query, { type: 'all' })} scroll={false} selected={query.type === 'all'}>
						<span>
							Все
							<span className='hidden md:inline'>
								{' · '}
								{rooms.length} {plural(rooms.length, ['тип', 'типа', 'типов'])}
							</span>
						</span>
					</Chip>
					{filters.map((filter) => (
						<Chip
							key={filter.type}
							href={roomsHref(query, { type: filter.type })}
							scroll={false}
							selected={query.type === filter.type}
						>
							{filter.label}
						</Chip>
					))}
					<Chip
						href={roomsHref(query, { freeOnly: !query.freeOnly })}
						scroll={false}
						selected={query.freeOnly}
						className='hidden md:inline-flex'
					>
						Только свободные
					</Chip>
				</div>
			</section>

			<section className='max-w-page mx-auto grid w-full gap-2 px-3 pt-3.5 md:grid-cols-2 md:gap-3 md:px-6 md:pt-6 lg:grid-cols-3'>
				{visible.length > 0 ? (
					visible.map((room, index) => (
						<RoomCard
							first={index === 0}
							key={room.id}
							room={room}
							availability={availability ? availability[room.id] : null}
							nights={query.nights}
						/>
					))
				) : (
					<p className='text-text-muted md:col-span-3' role='status'>
						По этим условиям номеров нет.{' '}
						<AppLink
							href={roomsHref(query, { type: 'all', freeOnly: false })}
							className='tap-area text-text font-semibold underline'
						>
							Показать все номера
						</AppLink>
					</p>
				)}
			</section>

			<Section className='grid gap-2 pt-10 md:grid-cols-3 md:gap-3 md:pt-[104px]'>
				<div className={cardClass('surface', 'rounded-lg p-5 md:col-span-2 md:rounded-xl md:p-9')}>
					<h2 className='heading mb-3 text-[22px] md:mb-5 md:text-4xl'>{content.text('hotel.amenities-heading')}</h2>
					<ul className='hidden flex-wrap gap-2 md:flex'>
						{amenities.map((item) => (
							<li
								key={item}
								className={cn(chipLi, item.startsWith('Wi-Fi') && 'bg-accent text-on-accent font-semibold')}
							>
								{item}
							</li>
						))}
					</ul>
					<ul className='flex flex-wrap gap-1.5 md:hidden'>
						{mobileAmenities.map((item) => (
							<li
								key={item}
								className={cn(chipLi, item.startsWith('Wi-Fi') && 'bg-accent text-on-accent font-semibold')}
							>
								{item}
							</li>
						))}
					</ul>
				</div>

				<div className={cardClass('inverse', 'hidden gap-2.5 rounded-xl p-9 md:flex')}>
					<h2 className='heading mb-1.5 text-[26px]'>{content.text('hotel.quick-facts-heading')}</h2>
					{quickFacts.map((fact, index) => (
						<p key={fact} className={index === quickFacts.length - 1 ? 'text-inverse-muted' : undefined}>
							{fillPlaceholders(fact, values)}
						</p>
					))}
					<AppLink href='/hotel' className='tap-area text-accent mt-auto font-semibold underline'>
						Все правила ↗
					</AppLink>
				</div>
			</Section>

			<section className='max-w-page mx-auto hidden w-full px-6 pt-6 pb-6 md:block'>
				<div className='bg-accent text-on-accent flex flex-wrap items-center justify-between gap-5 rounded-xl px-10 py-8'>
					<h2 className='heading text-[28px] tracking-[-0.02em]'>Удобнее по телефону?</h2>
					<div className='flex flex-wrap gap-2'>
						<Button href={site.phones.hotel.href} variant='inverse' size='xl' className='min-h-[50px]'>
							{site.phones.hotel.display}
						</Button>
						<Button href={site.phones.free.href} variant='surface' size='xl' className='min-h-[50px] border-0'>
							{site.phones.free.display} · бесплатно
						</Button>
					</div>
				</div>
			</section>
		</main>
	)
}
