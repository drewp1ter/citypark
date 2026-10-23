import type { LinkProps } from '@tanstack/react-router'

export type NavItem = { label: string; to: NonNullable<LinkProps['to']> }

export const mainNavItems: NavItem[] = [
	{ label: 'Заказы', to: '/orders' },
	{ label: 'Доставка', to: '/delivery' },
	{ label: 'Меню', to: '/menu' },
]

export const contentNavItems: NavItem[] = [
	{ label: 'Номера', to: '/content/rooms' },
	{ label: 'Новости', to: '/content/news' },
	{ label: 'Главная', to: '/content/home' },
	{ label: 'Отель', to: '/content/hotel' },
	{ label: 'Настройки сайта', to: '/content/settings' },
	{ label: 'Цены отеля', to: '/content/hotel-prices' },
]

export const tailNavItems: NavItem[] = [
	{ label: 'Свободные номера', to: '/free-rooms' },
	{ label: 'Отчёты', to: '/reports' },
]
