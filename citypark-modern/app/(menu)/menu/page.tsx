import { socialMetadata } from '@/shared/config'
import type { Metadata } from 'next'
import { MenuPage } from '@/pages/menu'

export const metadata: Metadata = {
	...socialMetadata(
		'Доставка — Сити Парк',
		'Меню доставки Сити Парк в Белореченске: блюда, описание и цены.',
		'/menu',
		'/images/delivery-pasta.jpg'
	),
	alternates: { canonical: '/menu' },
	description: 'Меню доставки Сити Парк в Белореченске: блюда, описание и цены.',
	title: 'Доставка — Сити Парк',
}

// The menu reads ?category=… on the server (and will read live prices from the POS), so it is rendered per request.
export default async function Page({ searchParams }: PageProps<'/menu'>) {
	const { category } = await searchParams
	return <MenuPage categoryId={typeof category === 'string' ? category : undefined} />
}
