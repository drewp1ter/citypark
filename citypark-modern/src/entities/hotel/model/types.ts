export type HotelRule = { title: string; text: string }

export type GuestInfoSection = {
	title: string
	/** Text of the accordion item. Missing for most of them in the design (not written yet). */
	text?: string
}

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
