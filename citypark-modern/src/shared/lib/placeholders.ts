import { formatPrice, typo } from './format'

const loggedNames = new Set<string>()

/** Replaces {{name}} with formatPrice(values[name]). Unknown names (or non-numeric values) are left as they are and logged once (console.error). */
export function fillPlaceholders(text: string, values: Readonly<Record<string, unknown>>): string {
	const filled = text.replace(/\{\{\s*([a-zA-Z]+)\s*\}\}/g, (match, name: string) => {
		const value = Object.hasOwn(values, name) ? values[name] : undefined
		if (typeof value === 'number') return formatPrice(value)
		if (!loggedNames.has(name)) {
			loggedNames.add(name)
			console.error(`Unknown price placeholder: ${name}`)
		}
		return match
	})
	return typo(filled)
}
