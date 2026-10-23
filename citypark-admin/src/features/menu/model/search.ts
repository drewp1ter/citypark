export type MenuSearch = { category?: number; q?: string }

/** Reads the URL search params; every invalid value is dropped silently. */
export function parseMenuSearch(raw: Record<string, unknown>): MenuSearch {
	const search: MenuSearch = {}
	const category = typeof raw.category === 'string' ? Number(raw.category) : raw.category
	if (typeof category === 'number' && Number.isInteger(category) && category >= 0) search.category = category
	// The router turns "123" into the number 123.
	const text = typeof raw.q === 'number' ? String(raw.q) : raw.q
	if (typeof text === 'string' && text.trim() !== '' && text.length <= 100) search.q = text
	return search
}
