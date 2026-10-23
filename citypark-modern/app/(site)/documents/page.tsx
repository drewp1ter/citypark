import type { Metadata } from 'next'
import { DocumentsPage } from '@/pages/legal'

export const metadata: Metadata = { title: 'Документы — Сити Парк' }

export default function Page() {
	return <DocumentsPage />
}
