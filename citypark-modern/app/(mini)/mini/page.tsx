import type { Metadata } from 'next'
import { MiniMenuPage } from '@/pages/mini'

export const metadata: Metadata = { title: 'Меню — Сити Парк' }

export default async function Page({ searchParams }: { searchParams: Promise<{ category?: string | string[] }> }) {
	const { category } = await searchParams
	return <MiniMenuPage categoryId={typeof category === 'string' ? category : undefined} />
}
