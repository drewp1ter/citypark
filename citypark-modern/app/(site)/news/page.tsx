import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'

export const metadata: Metadata = {
	...socialMetadata(
		'Новости — Сити Парк',
		'Новости отеля и ресторана Сити Парк в Белореченске.',
		'/news',
		'/images/hotel-building.jpg'
	),
	alternates: { canonical: '/news' },
	description: 'Новости отеля и ресторана Сити Парк в Белореченске.',
	title: 'Новости — Сити Парк',
}

export { NewsPage as default } from '@/pages/news'
