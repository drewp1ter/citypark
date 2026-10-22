import type { Metadata } from 'next'
import { LegalPage } from '@/pages/legal'

export const metadata: Metadata = { title: 'Оферта — Сити Парк' }

export default function Page() {
	return <LegalPage title='Оферта' />
}
