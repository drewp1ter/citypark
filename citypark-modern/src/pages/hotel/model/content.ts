// Copy that only the hotel page uses, taken from design/reference/*/hotel.html.

export const breakfast = {
	note: 'Заказ до 22:00 · доставка {{breakfastDelivery}}',
} as const

export const extras = {
	balconies: {
		title: 'Балконы над парком',
		text: 'В номерах 201 и 301 — просторные балконы с плетёными креслами ручной работы и видом на кроны деревьев.',
	},
	free: {
		title: 'Бесплатно',
		text: 'Wi-Fi, парковка, побудка, аптечка, кипяток, посуда, иголки и нитки.',
	},
} as const

/** The mobile page has its own, shorter wording for the check-in rule. */
export const mobileCheckInRule =
	'Заезд круглосуточно, выезд до 12:00. Поздний выезд: до 6 ч — {{lateCheckoutPerHour}}/час.'

/** Shown inside an information section whose text has not been written yet. */
export const emptySectionText = 'Скоро добавим.'
