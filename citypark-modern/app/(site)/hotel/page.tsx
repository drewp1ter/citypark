import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'

export const metadata: Metadata = {
	...socialMetadata(
		'Отель-люкс — Сити Парк',
		'Отель Сити Парк в Белореченске: информация об отеле и проживании.',
		'/hotel',
		'/images/hotel-building.jpg'
	),
	alternates: { canonical: '/hotel' },
	description: 'Отель Сити Парк в Белореченске: информация об отеле и проживании.',
	title: 'Отель-люкс — Сити Парк',
}

export { HotelPage as default } from '@/pages/hotel'
