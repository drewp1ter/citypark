import { productionOrigin, site, type SiteSettings } from '@/shared/config'
import type { WorkingHours } from '@/shared/lib'

/** Describe the same business facts rendered by ContactsPage, including its existing fallbacks. */
export function businessJsonLd(settings: SiteSettings, hours: WorkingHours): string {
	const address = {
		'@type': 'PostalAddress',
		streetAddress: settings.address.street,
		addressLocality: settings.address.city,
		postalCode: settings.address.postcode,
	}
	const data = {
		'@context': 'https://schema.org',
		'@graph': [
			{
				'@type': 'Hotel',
				'@id': `${productionOrigin}/hotel#business`,
				url: `${productionOrigin}/hotel`,
				name: site.name,
				address,
				telephone: settings.phones.hotel.href.slice(4),
				email: settings.email,
			},
			{
				'@type': 'Restaurant',
				'@id': `${productionOrigin}/restaurant#business`,
				url: `${productionOrigin}/restaurant`,
				name: site.name,
				address,
				telephone: settings.phones.restaurant.href.slice(4),
				email: settings.email,
				openingHours: `Mo-Su ${hours.start}-${hours.end}`,
			},
		],
	}
	// Prevent backend text from terminating the script element; JSON.parse restores the original values.
	return JSON.stringify(data).replace(/</g, '\\u003c')
}
