import type { MetadataRoute } from 'next'

// Browsers read the manifest outside our CSS, so the colours are raw hex copied from tokens.css.
export default function manifest(): MetadataRoute.Manifest {
	return {
		name: 'Сити Парк — отель, ресторан и доставка',
		short_name: 'Сити Парк',
		description:
			'Отель-люкс на 12 номеров, ресторан и летнее кафе, банкеты и доставка еды. Белореченск, ул. Гоголя, 61.',
		lang: 'ru',
		start_url: '/',
		scope: '/',
		display: 'standalone',
		background_color: '#f4f4f1', // --bg (light)
		theme_color: '#f4f4f1', // --bg (light)
		icons: [
			{ src: '/pwa/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
			{ src: '/pwa/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
			{ src: '/pwa/icon-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
		],
		shortcuts: [
			{ name: 'Меню и доставка', url: '/menu' },
			{ name: 'Номера', url: '/rooms' },
		],
	}
}
