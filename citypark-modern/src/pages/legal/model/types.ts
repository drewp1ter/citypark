export type LegalItem = {
	/** Text of the item. Plain text; `typo()` is applied when rendering. */
	text: string
	/** Paragraph under the item's text, before its sub-items. */
	note?: string
	items?: readonly LegalItem[]
}

export type LegalDocument = {
	title: string
	/** Paragraphs before the numbered list. */
	intro: readonly string[]
	items: readonly LegalItem[]
}
