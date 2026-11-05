import type { Metadata } from 'next'
import { DocumentsPage } from '@/pages/legal'

export const metadata: Metadata = { alternates: { canonical: '/documents' }, description: 'Документы Сити Парк в Белореченске.', title: 'Документы — Сити Парк' }

export default function Page() {
	return <DocumentsPage />
}
