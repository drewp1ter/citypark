const NAMED_ENTITIES: Record<string, string> = {
	amp: '&',
	lt: '<',
	gt: '>',
	quot: '"',
	apos: "'",
	nbsp: ' ',
}

function decodeEntities(text: string): string {
	return text.replace(
		/&(?:#(\d+)|#[xX]([0-9a-fA-F]+)|([a-zA-Z]+));/g,
		(whole, dec?: string, hex?: string, name?: string) => {
			if (name !== undefined) return NAMED_ENTITIES[name] ?? whole
			const code = dec !== undefined ? parseInt(dec, 10) : parseInt(hex ?? '', 16)
			return Number.isFinite(code) && code <= 0x10ffff ? String.fromCodePoint(code) : whole
		}
	)
}

/** "<p></p>", "<p><br></p>" and whitespace-only → ""; otherwise unchanged. */
export function normalizeHtml(html: string): string {
	const rest = html.replace(/<\/?p\s*>|<br\s*\/?>|&nbsp;|\s+/gi, '')
	return rest === '' ? '' : html
}

/** Visible character count, same idea as the backend's TextLength (tags stripped, entities decoded). */
export function textLength(html: string): number {
	const text = decodeEntities(html.replace(/<[^>]*>/g, ''))
	return Array.from(text).length
}
