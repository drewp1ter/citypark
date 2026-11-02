export const autofillReasons = [
	'unsupported',
	'permission_denied',
	'position_unavailable',
	'location_timeout',
	'request_failed',
	'request_timeout',
	'invalid_response',
	'invalid',
	'configuration',
	'empty_results',
	'unsupported_region',
	'no_town',
	'ambiguous_town',
	'partial',
	'provider_network',
	'provider_timeout',
	'provider_http',
	'provider_json',
	'provider_shape',
	'delivery_list',
	'town_matching',
] as const
export type AutofillReason = (typeof autofillReasons)[number]
export type AutofillDetails = {
	stage: 'browser' | 'validation' | 'configuration' | 'provider' | 'delivery_list' | 'matching'
	surface?: 'web' | 'mini'
	elapsed_ms?: number
	http_status?: number
	browser_code?: number
	secure_context?: boolean
	town_count?: number
	candidate_count?: number
	locality_source?: 'city' | 'settlement'
}

/** Explicit field and value allowlist shared by capture and the final SDK filter. */
export function sanitizeAutofill(value: Record<string, unknown>): Record<string, string | number | boolean> {
	const result: Record<string, string | number | boolean> = {}
	const enums: Record<string, readonly string[]> = {
		stage: ['browser', 'validation', 'configuration', 'provider', 'delivery_list', 'matching'],
		surface: ['web', 'mini'],
		locality_source: ['city', 'settlement'],
	}
	for (const [key, values] of Object.entries(enums))
		if (typeof value[key] === 'string' && values.includes(value[key] as string)) result[key] = value[key] as string
	for (const key of ['elapsed_ms', 'http_status', 'browser_code', 'town_count', 'candidate_count'])
		if (
			typeof value[key] === 'number' &&
			Number.isFinite(value[key]) &&
			value[key] >= 0 &&
			value[key] <= (key === 'http_status' ? 599 : key === 'browser_code' ? 3 : 86400000)
		)
			result[key] = value[key]
	if (typeof value.secure_context === 'boolean') result.secure_context = value.secure_context
	return result
}
