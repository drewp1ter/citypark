/** A page number from the URL: a whole number from 1, anything else is dropped. */
export function parsePage(value: unknown): number | undefined {
	const page = typeof value === 'string' && value.trim() !== '' ? Number(value) : value
	return typeof page === 'number' && Number.isInteger(page) && page >= 1 ? page : undefined
}
