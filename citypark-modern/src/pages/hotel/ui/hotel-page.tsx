import { getPageContent } from '@/entities/content'
import { getGuestDocuments, getHotelPrices } from '@/entities/hotel'
import { getRooms } from '@/entities/room'
import type { GuestInfoCopy, TitledHtml } from '../model/types'
import { Extras } from './extras'
import { GuestInfo } from './guest-info'
import { Hero } from './hero'
import { RoomFund } from './room-fund'
import { Rules } from './rules'

const RULES = 6
const GUEST_INFO_SECTIONS = 6

export async function HotelPage() {
	const [content, rooms, documents, prices] = await Promise.all([
		getPageContent('hotel'),
		getRooms().catch((error: unknown) => {
			console.error(error)
			return null
		}),
		getGuestDocuments(),
		getHotelPrices(),
	])
	const values = {
		breakfastDelivery: prices.breakfastDelivery,
		lateCheckoutPerHour: prices.lateCheckoutPerHour,
		smokingFine: prices.smokingFine,
	}
	const numbered = (name: string, count: number): TitledHtml[] =>
		Array.from({ length: count }, (_, index) => ({
			title: content.text(`hotel.${name}.${index + 1}.title`),
			html: content.html(`hotel.${name}.${index + 1}.text`, values),
		}))
	const guestInfo: GuestInfoCopy = {
		heading: content.text('hotel.guest-info.heading'),
		intro: content.text('hotel.guest-info.intro'),
		sections: numbered('guest-info', GUEST_INFO_SECTIONS),
	}

	return (
		<main>
			<Hero content={content} />
			{rooms && <RoomFund heading={content.text('hotel.fund.heading')} rooms={rooms} />}
			<Extras content={content} prices={prices} />
			<Rules
				heading={content.text('hotel.rules.heading')}
				rules={numbered('rules', RULES)}
				mobileCheckInRule={content.text('hotel.rules.check-in-mobile', values)}
				guestInfo={guestInfo}
				documents={documents}
			/>
			<GuestInfo copy={guestInfo} documents={documents} />
		</main>
	)
}
