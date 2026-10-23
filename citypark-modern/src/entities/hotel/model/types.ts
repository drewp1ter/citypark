export type GuestDocument = {
	title: string
	/** "PDF · 2 стр." */
	meta: string
	/** Link to the file. The PDFs have not been sent yet. */
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
