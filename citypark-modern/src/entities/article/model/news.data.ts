import { PLACEHOLDER_PHOTO } from '@/shared/lib'
import type { BlogAuthor, BlogPost, NewsItem } from './types'

// Copied from design/reference/*/news.html and home.html. Newest first.
// Three photos are missing (the owner will send them): those items use the placeholder image.
export const news: readonly NewsItem[] = [
	{
		slug: 'special-offer-september-2026',
		date: '2026-09-07',
		title: 'Спецпредложение сентября',
		excerpt: 'Что входит, цена и срок действия.',
		photo: { src: PLACEHOLDER_PHOTO, alt: 'Спецпредложение сентября' },
	},
	{
		slug: 'special-offer-june-2026',
		date: '2026-06-24',
		title: 'Спецпредложение июня',
		photo: { src: PLACEHOLDER_PHOTO, alt: '' },
	},
	{
		slug: 'special-offer-may-2026',
		date: '2026-05-07',
		title: 'Спецпредложение мая',
		photo: { src: PLACEHOLDER_PHOTO, alt: '' },
	},
	{
		slug: 'new-menu-2026',
		date: '2026-03-16',
		title: 'Новое меню в Сити Парке 2026',
		shortTitle: 'Новое меню 2026',
		photo: { src: '/images/news-menu.jpg', alt: '' },
	},
	{
		slug: 'new-steaks-2025',
		date: '2025-06-30',
		title: 'Новые стейки',
		photo: { src: '/images/news-steaks.jpg', alt: '' },
	},
	{
		slug: 'updated-menu-2025',
		date: '2025-05-01',
		title: 'Обновлённое меню',
		photo: { src: '/images/news-update.jpg', alt: '' },
	},
	{
		slug: 'lenten-menu-2025',
		date: '2025-03-02',
		title: 'Постное меню 2025',
		photo: { src: PLACEHOLDER_PHOTO, alt: '' },
	},
]

export const blogAuthor: BlogAuthor = {
	name: 'Денис Троицкий',
	photo: { src: '/images/chef.jpg', alt: 'Шеф-повар Денис Троицкий' },
}

export const blogPosts: readonly BlogPost[] = [
	{
		slug: 'frittata',
		title: 'Фриттата',
		subtitle: 'Итальянский омлет',
		photo: { src: '/images/blog-frittata.jpg', alt: 'Фриттата' },
	},
	{
		slug: 'pasta-chicken-spinach',
		title: 'Паста с курицей и шпинатом',
		subtitle: 'Просто и полезно',
		shortTitle: 'Паста с курицей',
		photo: { src: '/images/blog-pasta.jpg', alt: 'Паста с курицей и шпинатом' },
	},
	{
		slug: 'lunch-set-fish',
		title: 'Обеденный сет с рыбой',
		subtitle: 'Свежесть и сытность',
		shortTitle: 'Сет с рыбой',
		photo: { src: '/images/blog-lunch-set-fish.jpg', alt: 'Обеденный сет с рыбой' },
	},
]
