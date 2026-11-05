import type { Metadata } from 'next'
import { LegalPage, privacyPolicy } from '@/pages/legal'

export const metadata: Metadata = { alternates: { canonical: '/privacy' }, description: 'Политика конфиденциальности Сити Парк.', title: `${privacyPolicy.title} — Сити Парк` }

export default function Page() {
	return <LegalPage document={privacyPolicy} />
}
