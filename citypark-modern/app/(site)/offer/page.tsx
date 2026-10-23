import type { Metadata } from 'next'
import { LegalPage, publicOffer } from '@/pages/legal'

export const metadata: Metadata = { title: `${publicOffer.title} — Сити Парк` }

export default function Page() {
	return <LegalPage document={publicOffer} />
}
