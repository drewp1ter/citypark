'use client'

import { usePathname } from 'next/navigation'
import { site } from '@/shared/config'
import { IconButton, PhoneIcon } from '@/shared/ui'

/** Header phone button. Hotel pages call the hotel, everything else calls the restaurant. */
export function PhoneButton() {
	const pathname = usePathname() ?? ''
	const hotel = pathname.startsWith('/rooms') || pathname.startsWith('/hotel')

	return (
		<IconButton
			href={hotel ? site.phones.hotel.href : site.phones.restaurant.href}
			aria-label={hotel ? 'Позвонить в отель' : 'Позвонить'}
		>
			<PhoneIcon />
		</IconButton>
	)
}
