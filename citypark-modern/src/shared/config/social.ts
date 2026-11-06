import type { Metadata } from 'next'
import { productionOrigin } from './search'

export function socialMetadata(
	title: string,
	description: string,
	path: string,
	image = '/images/hotel-building.jpg',
	alt = 'Сити Парк',
	type: 'website' | 'article' = 'website'
): Metadata {
	const url = new URL(path, productionOrigin).href
	const src = new URL(
		!image || image.endsWith('.svg') || image.startsWith('http://') ? '/images/hotel-building.jpg' : image,
		productionOrigin
	).href
	return {
		openGraph: { title, description, url, siteName: 'Сити Парк', locale: 'ru_RU', type, images: [{ url: src, alt }] },
		twitter: { card: 'summary_large_image', title, description, images: [{ url: src, alt }] },
	}
}
