import { typo } from '@/shared/lib'
import type { GuestDocument, GuestInfoSection, HotelRule } from './types'

// Copied from design/reference/*/hotel.html and rooms.html.

/** "В каждом номере" */
export const roomAmenities: readonly string[] = [
	'Сплит-система',
	'Smart TV',
	'Сейф',
	'Холодильник',
	'Шкаф-гардероб',
	'Комод',
	'Журнальный столик',
	'Пуф',
	'Внутренний телефон',
	'Душевая кабина',
	'Фен',
	'Wi-Fi бесплатно',
]

/** "Коротко" on the rooms page. */
export const quickFacts: readonly string[] = [
	'Заезд 24/7, выезд до 12:00',
	'Наличные или карта, без предоплаты',
	'Без животных · не курить (штраф {{smokingFine}})',
]

/** "Правила" on the hotel page. */
export const rules: readonly HotelRule[] = [
	{
		title: 'Заезд и выезд',
		text: typo('Заезд круглосуточно, расчётный час 12:00. Ранний заезд с 8:00 — без доплаты.'),
	},
	{
		title: 'Поздний выезд',
		text: 'До 6 ч — {{lateCheckoutPerHour}}/час · 6–12 ч — половина суток · 12–24 ч — сутки.',
	},
	{
		title: 'Оплата',
		text: 'Наличные или карта. Бронирование бесплатно, снимается при опоздании более чем на сутки.',
	},
	{ title: 'Гости', text: 'VIP — до 4, остальные — до 2. Заселение по паспорту.' },
	{
		title: 'Дети до 14 лет',
		text: 'С родителями или законными представителями и свидетельством о рождении.',
	},
	{ title: 'Нельзя', text: 'С животными. Курить в помещениях — штраф {{smokingFine}}.' },
]

/** "Информация для гостей" on the hotel page. */
export const guestInfo = {
	intro: typo('По Правилам предоставления гостиничных услуг (ПП РФ № 1085)'),
	sections: [
		{
			title: 'Об исполнителе',
			text: typo(
				'ООО «Олимп», гостиница «Сити Парк». 352630, Краснодарский край, г. Белореченск, ул. Гоголя, 61. Тел. (86155) 2-25-38. Категория «Без звёзд».'
			),
		},
		{ title: typo('Категории номеров и цены 2026') },
		{ title: 'Завтраки и лобби-бар' },
		{ title: 'Бронирование и аннуляция' },
		{ title: 'Правила проживания и пожарной безопасности' },
		{ title: 'Порча имущества' },
	] satisfies readonly GuestInfoSection[],
	documents: [
		{ title: 'Выписка из реестра объектов классификации', meta: typo('PDF · 2 стр.') },
		{ title: 'Политика обработки персональных данных', meta: typo('PDF · 8 стр.') },
		{ title: 'Публичная оферта', meta: 'PDF' },
	] satisfies readonly GuestDocument[],
} as const
