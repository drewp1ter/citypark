const money = new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 0, maximumFractionDigits: 2 })

/** Quotes may contain kopecks; the menu's whole-ruble formatter would hide them. */
export function formatQuotePrice(value: number): string {
	return `${money.format(value)}\u00a0₽`
}
