import type { Metadata } from 'next'
import { LegalPage, privacyPolicy } from '@/pages/legal'

export const metadata: Metadata = { title: `${privacyPolicy.title} — Сити Парк` }

export default function Page() {
	return <LegalPage document={privacyPolicy} />
}
