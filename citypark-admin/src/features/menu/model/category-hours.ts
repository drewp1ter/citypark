export const WEEKDAYS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'] as const

export function categoryDays(salesDays: string): boolean[] {
	return WEEKDAYS.map((day) => salesDays === '' || salesDays.split(', ').includes(day))
}
