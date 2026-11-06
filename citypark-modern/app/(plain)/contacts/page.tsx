import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'

export const metadata: Metadata = {
	...socialMetadata(
		'Контакты — Сити Парк',
		'Адрес и телефоны отеля, ресторана и доставки Сити Парк в Белореченске.',
		'/contacts',
		'/images/hotel-building.jpg'
	),
	alternates: { canonical: '/contacts' },
	description: 'Адрес и телефоны отеля, ресторана и доставки Сити Парк в Белореченске.',
	title: 'Контакты — Сити Парк',
}

export { ContactsPage as default } from '@/pages/contacts'
