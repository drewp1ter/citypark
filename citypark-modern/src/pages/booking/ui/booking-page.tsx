import type { Metadata } from 'next'
import { getRooms, bookingRoomIds, bookingUuid } from '@/entities/room'
import { getSiteSettings } from '@/shared/api'
import { addDays, todayInMoscow } from '@/shared/lib'
import { Breadcrumbs, Button } from '@/shared/ui'
import { BookingForm } from './booking-form'

export const metadata: Metadata = { title: 'Бронирование — Сити Парк', robots: { index: false, follow: false } }
export async function BookingPage({
	searchParams,
}: {
	searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
	const params = await searchParams
	const first = (key: string) => (typeof params[key] === 'string' ? (params[key] as string) : undefined)
	const [rooms, settings] = await Promise.all([getRooms().catch(() => null), getSiteSettings()])
	const checkIn = first('checkIn') ?? todayInMoscow()
	const key = bookingUuid.safeParse(first('attempt'))
	return (
		<main className='max-w-page mx-auto w-full px-3 pt-4 pb-20 md:px-6 md:pt-8'>
			<Breadcrumbs items={[{ label: 'Номера', href: '/rooms' }, { label: 'Бронирование' }]} />
			<h1 className='heading mb-6 text-[32px] md:text-5xl'>Бронирование номера</h1>
			{rooms ? (
				<BookingForm
					rooms={rooms.filter((r) => bookingRoomIds.some((id) => id === r.id))}
					initialStay={{
						roomTypeId: bookingRoomIds.find((id) => id === first('roomTypeId')) ?? 'double-twin',
						checkIn,
						checkOut: first('checkOut') ?? addDays(checkIn, 1),
						guests: Number(first('guests') ?? 2),
					}}
					initialKey={key.success ? key.data : undefined}
					hotelPhone={settings.phones.hotel}
				/>
			) : (
				<p role='alert'>Не удалось загрузить номера. Позвоните нам.</p>
			)}
			<div className='mt-8 flex flex-wrap items-center gap-3'>
				<span>Связаться с отелем:</span>
				<Button href={settings.phones.hotel.href} variant='outline'>
					{settings.phones.hotel.display}
				</Button>
			</div>
		</main>
	)
}
