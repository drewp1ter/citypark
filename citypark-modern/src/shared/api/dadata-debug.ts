/** Opt-in local diagnostics. Never send provider data to the browser or production logs. */
export function dadataDebug(event: string, details: unknown): void {
	if (process.env.DADATA_DEBUG !== '1') return
	const key = process.env.DADATA_API_KEY
	const serialized = JSON.stringify(details, null, 2)
	const output = key ? serialized.replaceAll(JSON.stringify(key).slice(1, -1), '[REDACTED]') : serialized
	console.info(`[DaData geolocation] ${event}\n${output}`)
}
