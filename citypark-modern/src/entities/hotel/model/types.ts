export type GuestDocument = {
	title: string
	/** "2 стр." */
	meta: string
	/** Page with the document: scans or text. */
	href?: string
}

export type BreakfastItem = {
	name: string
	/** Name on mobile. Empty when the full name is used everywhere. */
	shortName: string
	price: number
	/** Hidden on mobile. */
	desktopOnly: boolean
}

/** The hotel's fees and the breakfast menu, in rubles. */
export type HotelPrices = {
	breakfastDelivery: number
	lateCheckoutPerHour: number
	smokingFine: number
	breakfast: readonly BreakfastItem[]
}
