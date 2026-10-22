import { getGuestInfo, getHotelRules } from '@/entities/hotel'
import { getRooms } from '@/entities/room'
import { Extras } from './extras'
import { GuestInfo } from './guest-info'
import { Hero } from './hero'
import { RoomFund } from './room-fund'
import { Rules } from './rules'

export async function HotelPage() {
	const [rooms, rules, guestInfo] = await Promise.all([getRooms(), getHotelRules(), getGuestInfo()])

	return (
		<main>
			<Hero />
			<RoomFund rooms={rooms} />
			<Extras />
			<Rules rules={rules} guestInfo={guestInfo} />
			<GuestInfo intro={guestInfo.intro} sections={guestInfo.sections} documents={guestInfo.documents} />
		</main>
	)
}
