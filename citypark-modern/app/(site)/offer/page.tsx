import type { Metadata } from 'next'
import { LegalPage, publicOffer } from '@/pages/legal'

export const metadata: Metadata = { alternates: { canonical: '/offer' }, description: 'Публичная оферта Сити Парк.', title: `${publicOffer.title} — Сити Парк` }

export default function Page() {
	return <LegalPage document={publicOffer} />
}
