import type { RoomKind } from './types'

const tours: Record<RoomKind, string> = { single: 'room_1', double: 'room_2', vip: 'vip_1' }

/** Old site's Pano2VR tour of a room type, served from public/tour. */
export function roomTourHref(kind: RoomKind): string {
	return `/tour/${tours[kind]}.html`
}
