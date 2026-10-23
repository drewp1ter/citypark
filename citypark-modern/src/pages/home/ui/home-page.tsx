import { getNews } from '@/entities/article'
import { getMenu } from '@/entities/dish'
import { getFreeRoomsCount, getRooms } from '@/entities/room'
import { PLACEHOLDER_PHOTO } from '@/shared/lib'
import { getPageContent } from '../api/get-page-content'
import { ContactsBlock } from './contacts-block'
import { DeliverySection } from './delivery-section'
import { Hero } from './hero'
import { NewsSection } from './news-section'
import { RestaurantSection } from './restaurant-section'
import { RoomsSection } from './rooms-section'

const HOME_CATEGORIES = 4
const HOME_DISHES = 3

export async function HomePage() {
	const [content, rooms, freeRooms, menu, news] = await Promise.all([
		getPageContent('home'),
		getRooms(),
		getFreeRoomsCount().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getMenu().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getNews(),
	])
	const available = menu?.dishes.filter((dish) => dish.available) ?? []
	const withPhoto = available.filter((dish) => dish.photo.src !== PLACEHOLDER_PHOTO)
	const dishes = [...withPhoto, ...available.filter((dish) => !withPhoto.includes(dish))].slice(0, HOME_DISHES)
	const latest = news[0]
	const second = news.find((item) => item.slug === 'new-menu-2026') ?? news[1]
	const third = news.find((item) => item !== latest && item !== second) ?? news[2]

	return (
		<main>
			<Hero content={content} rooms={rooms} />
			<RoomsSection content={content} rooms={rooms} free={freeRooms} />
			<RestaurantSection content={content} />
			{menu && (
				<DeliverySection content={content} categories={menu.categories.slice(0, HOME_CATEGORIES)} dishes={dishes} />
			)}
			<NewsSection title={content.text('home.news.heading')} items={[latest, second, third]} />
			<div className='max-w-page mx-auto w-full px-3 pt-14 pb-0 md:px-6 md:pt-28 md:pb-6'>
				<ContactsBlock />
			</div>
		</main>
	)
}
