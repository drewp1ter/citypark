import { getNews } from '@/entities/article'
import { getHomepageMenuConfiguration, getMenu, selectHomepageMenu } from '@/entities/dish'
import { getFreeRoomsCount, getRooms } from '@/entities/room'
import { PLACEHOLDER_PHOTO } from '@/shared/lib'
import { getPageContent } from '@/entities/content'
import { ContactsBlock } from './contacts-block'
import { DeliverySection } from './delivery-section'
import { Hero } from './hero'
import { NewsSection } from './news-section'
import { RestaurantSection } from './restaurant-section'
import { RoomsSection } from './rooms-section'

export async function HomePage() {
	const [content, rooms, freeRooms, menu, news, homepageMenu] = await Promise.all([
		getPageContent('home'),
		getRooms().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getFreeRoomsCount().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getMenu().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getNews().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getHomepageMenuConfiguration().catch((error: unknown) => {
			console.error(error)
			return { categories: [] }
		}),
	])
	const groups = menu ? selectHomepageMenu(menu, homepageMenu, PLACEHOLDER_PHOTO) : []
	const latestNews = news && news.length >= 3 ? ([news[0], news[1], news[2]] as const) : null

	return (
		<main>
			<Hero content={content} rooms={rooms ?? []} />
			<RoomsSection content={content} rooms={rooms} free={freeRooms} />
			<RestaurantSection content={content} />
			{groups.length > 0 && <DeliverySection content={content} groups={groups} />}
			{latestNews && <NewsSection title={content.text('home.news.heading')} items={latestNews} />}
			<div className='max-w-page mx-auto w-full px-3 pt-14 pb-0 md:px-6 md:pt-28 md:pb-6'>
				<ContactsBlock />
			</div>
		</main>
	)
}
