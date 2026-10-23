'use client'

import { usePathname } from 'next/navigation'
import { useSiteSettings } from '@/shared/config'
import { IconButton, PhoneIcon } from '@/shared/ui'

/** Header phone button. Hotel pages call the hotel, everything else calls the restaurant. */
export function PhoneButton() {
	const { phones } = useSiteSettings()
	const pathname = usePathname() ?? ''
	const hotel = pathname.startsWith('/rooms') || pathname.startsWith('/hotel')

	return (
		<IconButton
			href={hotel ? phones.hotel.href : phones.restaurant.href}
			aria-label={hotel ? 'Позвонить в отель' : 'Позвонить'}
		>
			<PhoneIcon />
		</IconButton>
	)
}
