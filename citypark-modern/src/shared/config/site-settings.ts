import { nb } from '@/shared/lib'

export type SitePhone = {
	/** Label in the contacts block and in the mobile menu. */
	label: string
	/** Short label for narrow layouts. */
	shortLabel: string
	display: string
	href: string
	/** A second (landline) number shown on the contacts page. */
	alt?: { display: string; href: string }
}

export type SiteSettings = {
	phones: { restaurant: SitePhone; hotel: SitePhone; free: SitePhone; landline: SitePhone }
	address: { postcode: string; city: string; street: string; full: string }
	email: string
	company: { name: string; inn: string; ogrn: string }
	social: readonly { href: string; label: string; name: string; display: string; handle: string }[]
	/** Search link to the address on Yandex Maps, used by the "Маршрут" button. */
	mapUrl: string
}

type ApiPhone = {
	label: string
	shortLabel: string
	display: string
	number: string
	altDisplay: string
	altNumber: string
}

/** The body of `GET /settings`. */
export type ApiSiteSettings = {
	phones: { restaurant: ApiPhone; hotel: ApiPhone; free: ApiPhone; landline: ApiPhone }
	address: { postcode: string; city: string; street: string }
	email: string
	company: { name: string; inn: string; ogrn: string }
	social: readonly { name: string; label: string; url: string; display: string }[]
	updatedAt: string
}

/** The contacts the site had before they moved to the backend. Shown when the backend does not answer. */
export const defaultSiteSettings: SiteSettings = {
	phones: {
		restaurant: {
			label: 'Ресторан и доставка',
			shortLabel: 'Ресторан',
			display: nb('8 918 311-97-91'),
			href: 'tel:+79183119791',
			alt: { display: nb('8 (86155) 3-30-03'), href: 'tel:+78615533003' },
		},
		hotel: {
			label: 'Отель',
			shortLabel: 'Отель',
			display: nb('8 918 311-97-10'),
			href: 'tel:+79183119710',
			alt: { display: nb('8 (86155) 3-30-02'), href: 'tel:+78615533002' },
		},
		free: {
			label: 'Бесплатно по России',
			shortLabel: 'По России',
			display: nb('8 800 100-24-41'),
			href: 'tel:88001002441',
		},
		landline: {
			label: 'Городской',
			shortLabel: 'Городской',
			display: nb('(86155) 2-25-38'),
			href: 'tel:+78615522538',
		},
	},
	address: {
		postcode: '352630',
		city: 'Белореченск',
		street: 'ул. Гоголя, 61',
		full: '352630, г. Белореченск, ул. Гоголя, 61',
	},
	email: 'belcitypark@mail.ru',
	company: {
		name: 'ООО «Олимп»',
		inn: '2309117400',
		ogrn: '1092309002378',
	},
	social: [
		{
			href: 'https://vk.com/cityparkvip',
			label: 'VK',
			name: 'ВКонтакте',
			display: 'vk.com/cityparkvip',
			handle: 'cityparkvip',
		},
		{
			href: 'https://ok.ru/cityparkvip',
			label: 'OK',
			name: 'Одноклассники',
			display: 'ok.ru/cityparkvip',
			handle: 'cityparkvip',
		},
	],
	mapUrl: `https://yandex.ru/maps/?text=${encodeURIComponent('Белореченск, ул. Гоголя, 61')}`,
}

function toSitePhone(phone: ApiPhone): SitePhone {
	return {
		label: phone.label,
		shortLabel: phone.shortLabel,
		display: nb(phone.display),
		href: `tel:${phone.number}`,
		alt:
			phone.altDisplay && phone.altNumber
				? { display: nb(phone.altDisplay), href: `tel:${phone.altNumber}` }
				: undefined,
	}
}

/** Turns the backend's contacts into the shape the site uses. */
export function toSiteSettings(api: ApiSiteSettings): SiteSettings {
	const { postcode, city, street } = api.address
	return {
		phones: {
			restaurant: toSitePhone(api.phones.restaurant),
			hotel: toSitePhone(api.phones.hotel),
			free: toSitePhone(api.phones.free),
			landline: toSitePhone(api.phones.landline),
		},
		address: { postcode, city, street, full: `${postcode}, г. ${city}, ${street}` },
		email: api.email,
		company: api.company,
		social: api.social.map((item) => ({
			href: item.url,
			label: item.label,
			name: item.name,
			display: item.display,
			handle: item.url.split('/').filter(Boolean).at(-1) ?? item.display,
		})),
		mapUrl: `https://yandex.ru/maps/?text=${encodeURIComponent(`${city}, ${street}`)}`,
	}
}
