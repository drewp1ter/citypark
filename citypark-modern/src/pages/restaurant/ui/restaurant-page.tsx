import { BookingSection } from './booking-section'
import { Banquets } from './banquets'
import { Hero } from './hero'
import { Venues } from './venues'

export function RestaurantPage() {
	return (
		<main>
			<Hero />
			<Venues />
			<Banquets />
			<BookingSection />
		</main>
	)
}
