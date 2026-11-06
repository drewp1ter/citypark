import type { MetadataRoute } from 'next'
import { getNews } from '@/entities/article'
import { getMenu } from '@/entities/dish'
import { getRooms } from '@/entities/room'
import { productionOrigin } from '@/shared/config'

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
	// Propagate failures: never publish an incomplete sitemap as a successful response.
	const [rooms, news, menu] = await Promise.all([getRooms(), getNews(), getMenu()])
	const paths = [
		'/',
		'/restaurant',
		'/hotel',
		'/rooms',
		'/menu',
		'/news',
		'/contacts',
		'/documents',
		'/offer',
		'/privacy',
		...menu.dishes.map((dish) => `/menu/${encodeURIComponent(dish.id)}`),
		...rooms.map((room) => `/rooms/${encodeURIComponent(room.id)}`),
		...news.map((article) => `/news/${encodeURIComponent(article.slug)}`),
	]
	return [...new Set(paths)].map((path) => ({ url: productionOrigin + path }))
}
