import type { Room } from '@/entities/room'
import { formatPrice } from '@/shared/lib'
import { AppLink } from '@/shared/ui'

// "Номерной фонд" lists the cheapest rooms first and the VIP room last.
const ORDER = ['single-balcony', 'single', 'double-twin', 'double-euro', 'vip']

/** Price list of the room types: a table row on desktop, a two-line row on mobile. */
export function RoomFund({ heading, rooms }: { heading: string; rooms: readonly Room[] }) {
	const sorted = ORDER.flatMap((id) => rooms.filter((room) => room.id === id))

	return (
		<section className='max-w-page mx-auto w-full px-3 pt-11 md:px-6 md:pt-[104px]'>
			<div className='mb-3 flex items-end justify-between gap-4 md:mb-6'>
				<h2 className='heading mx-1.5 text-[26px] tracking-[-0.02em] md:mx-0 md:text-5xl md:tracking-[-0.03em]'>
					{heading}
				</h2>
				<AppLink href='/rooms' className='tap-area font-semibold max-md:hidden'>
					Фото и бронь ↗
				</AppLink>
			</div>
			<ul className='border-border bg-surface rounded-md border px-4 py-1 md:rounded-lg md:px-6 md:py-2'>
				{sorted.map((room) => (
					<li key={room.id} className='border-divider border-b last:border-b-0'>
						<AppLink
							href={`/rooms/${room.id}`}
							className='flex items-center justify-between gap-2.5 py-3.5 md:grid md:grid-cols-[2fr_2fr_1.4fr_1fr] md:gap-4 md:py-[18px]'
						>
							<span className='flex flex-col md:contents'>
								<strong className='md:text-lg'>{room.hotelName ?? room.name}</strong>
								<span className='text-text-muted text-[13px] md:hidden'>
									{room.numbers} · {room.area}
								</span>
								<span className='text-text-muted hidden md:inline'>{room.hotelNote}</span>
								<span className='text-text-muted hidden md:inline'>{room.area}</span>
							</span>
							<span className='font-display font-medium whitespace-nowrap md:text-right md:text-[19px]'>
								{formatPrice(room.pricePerNight)}
							</span>
						</AppLink>
					</li>
				))}
			</ul>
		</section>
	)
}
