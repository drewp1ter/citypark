import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'
import { LegalPage, publicOffer } from '@/pages/legal'

export const metadata: Metadata = {
	...socialMetadata(
		`${publicOffer.title} — Сити Парк`,
		'Публичная оферта Сити Парк.',
		'/offer',
		'/images/hotel-building.jpg'
	),
	alternates: { canonical: '/offer' },
	description: 'Публичная оферта Сити Парк.',
	title: `${publicOffer.title} — Сити Парк`,
}

export default function Page() {
	return <LegalPage document={publicOffer} />
}
