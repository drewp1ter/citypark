import type { Metadata } from 'next'
import { SiteLayout } from '@/app/layouts'
import { NotFoundPage } from '@/pages/not-found'

export const metadata: Metadata = { title: 'Страница не найдена — Сити Парк' }

export default function NotFound() {
	return (
		<SiteLayout quickActions={false}>
			<NotFoundPage />
		</SiteLayout>
	)
}
