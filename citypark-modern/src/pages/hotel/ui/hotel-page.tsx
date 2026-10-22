import { getGuestInfo, getHotelPrices, getHotelRules } from '@/entities/hotel'
import { getRooms } from '@/entities/room'
import { fillPlaceholders } from '@/shared/lib'
import { mobileCheckInRule } from '../model/content'
import { Extras } from './extras'
import { GuestInfo } from './guest-info'
import { Hero } from './hero'
import { RoomFund } from './room-fund'
import { Rules } from './rules'

export async function HotelPage() {
	const [rooms, rules, guestInfo, prices] = await Promise.all([
		getRooms(),
		getHotelRules(),
		getGuestInfo(),
		getHotelPrices(),
	])
	const filledRules = rules.map((rule) => ({ ...rule, text: fillPlaceholders(rule.text, prices) }))

	return (
		<main>
			<Hero />
			<RoomFund rooms={rooms} />
			<Extras prices={prices} />
			<Rules
				rules={filledRules}
				mobileCheckInRule={fillPlaceholders(mobileCheckInRule, prices)}
				guestInfo={guestInfo}
			/>
			<GuestInfo intro={guestInfo.intro} sections={guestInfo.sections} documents={guestInfo.documents} />
		</main>
	)
}
