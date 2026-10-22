import { nb } from '@/shared/lib'

type Phone = {
	/** Label in the contacts block and in the mobile menu. */
	label: string
	/** Short label for narrow layouts. */
	shortLabel: string
	display: string
	href: string
	/** A second (landline) number shown on the contacts page. */
	alt?: { display: string; href: string }
}

const phones = {
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
} satisfies Record<string, Phone>

export const site = {
	name: 'Сити Парк',
	wordmark: 'Сити Парк',
	address: {
		postcode: '352630',
		city: 'Белореченск',
		street: 'ул. Гоголя, 61',
		full: '352630, г. Белореченск, ул. Гоголя, 61',
	},
	email: 'belcitypark@mail.ru',
	phones,
	company: {
		name: 'ООО «Олимп»',
		inn: '2309117400',
		ogrn: '1092309002378',
	},
	chef: 'Денис Троицкий',
	year: 2026,
	nav: [
		{ href: '/restaurant', label: 'Ресторан' },
		{ href: '/hotel', label: 'Отель' },
		{ href: '/rooms', label: 'Номера' },
		{ href: '/menu', label: 'Доставка' },
		{ href: '/news', label: 'Новости' },
		{ href: '/contacts', label: 'Контакты' },
	],
	legal: [
		{ href: '/documents', label: 'Документы' },
		{ href: '/offer', label: 'Оферта' },
		{ href: '/privacy', label: 'Конфиденциальность' },
	],
	/** Search link to the address on Yandex Maps, used by the "Маршрут" button. */
	mapUrl: `https://yandex.ru/maps/?text=${encodeURIComponent('Белореченск, ул. Гоголя, 61')}`,
	// Pages named on the contacts page of the design.
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
} as const
