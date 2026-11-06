import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'
import { parseRoomsQuery, RoomsPage } from '@/pages/rooms'

export const metadata: Metadata = {
	...socialMetadata(
		'Номера — Сити Парк',
		'Номера отеля Сити Парк в Белореченске: фотографии, описание и цены.',
		'/rooms',
		'/images/room-vip.jpg'
	),
	alternates: { canonical: '/rooms' },
	description: 'Номера отеля Сити Парк в Белореченске: фотографии, описание и цены.',
	title: 'Номера — Сити Парк',
}

// Dates, guests and filters come from the address, so the page is rendered per request
// (and will ask the booking engine for availability).
export default async function Page({ searchParams }: PageProps<'/rooms'>) {
	return <RoomsPage query={parseRoomsQuery(await searchParams)} />
}
