import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'
import { DocumentsPage } from '@/pages/legal'

export const metadata: Metadata = {
	...socialMetadata(
		'Документы — Сити Парк',
		'Документы Сити Парк в Белореченске.',
		'/documents',
		'/images/hotel-building.jpg'
	),
	alternates: { canonical: '/documents' },
	description: 'Документы Сити Парк в Белореченске.',
	title: 'Документы — Сити Парк',
}

export default function Page() {
	return <DocumentsPage />
}
