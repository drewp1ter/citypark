import type { Photo } from '@/shared/lib'

type BlockBase = {
	/** `<page>.<section>.<name>`, lowercase latin, digits and `-`. */
	key: string
	/** Russian section name staff recognise in the admin. */
	section: string
	/** Russian label shown in the admin. */
	label: string
	maxLength?: number
}

/** One editable piece of page copy. Mirrors the content blocks of citypark-go. */
export type Block = BlockBase &
	(
		| { type: 'text'; value: string }
		| { type: 'lines'; value: readonly string[] }
		| { type: 'richtext'; value: string }
		| { type: 'image'; image: Photo }
	)

export type ContentPage = 'home' | 'hotel'

export type PageContent = {
	/** Plain text with placeholders filled; '' when the key is missing. */
	text(key: string, values?: Readonly<Record<string, number>>): string
	lines(key: string): readonly string[]
	/** Sanitized HTML (the backend sanitizes; static data is trusted) with placeholders filled; '' when missing. */
	html(key: string, values?: Readonly<Record<string, number>>): string
	image(key: string): Photo | null
}
