import { socialMetadata } from '@/shared/config'
export const metadata = {
	...socialMetadata(
		'Сити Парк — отель, ресторан и доставка в Белореченске',
		'Отель-люкс на 12 номеров, ресторан и летнее кафе, банкеты и доставка еды. Белореченск, ул. Гоголя, 61.',
		'/',
		'/images/hotel-building.jpg'
	),
	alternates: { canonical: '/' },
}

export { HomePage as default } from '@/pages/home'
