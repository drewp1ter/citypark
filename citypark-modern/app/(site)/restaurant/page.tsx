import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'

export const metadata: Metadata = {
	...socialMetadata(
		'Ресторан и летнее кафе — Сити Парк',
		'Ресторан и летнее кафе в Белореченске. Сити Парк, ул. Гоголя, 61.',
		'/restaurant',
		'/images/restaurant-hall.jpg'
	),
	alternates: { canonical: '/restaurant' },
	description: 'Ресторан и летнее кафе в Белореченске. Сити Парк, ул. Гоголя, 61.',
	title: 'Ресторан и летнее кафе — Сити Парк',
}

export { RestaurantPage as default } from '@/pages/restaurant'
