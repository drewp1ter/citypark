export const site = {
	name: 'Сити Парк',
	wordmark: 'Сити Парк',
	chef: 'Денис Троицкий',
	year: 2026,
	map: { center: [39.8749086, 44.7596453] as const, zoom: 16 },
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
} as const
