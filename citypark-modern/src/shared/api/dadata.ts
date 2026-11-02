import { GeocodeError } from './geocode-error'
import { serverEnv } from '@/shared/config'
import { dadataDebug } from './dadata-debug'

const ENDPOINT = 'https://suggestions.dadata.ru/suggestions/api/4_1/rs/suggest/address'
const GEOLOCATE_ENDPOINT = 'https://suggestions.dadata.ru/suggestions/api/4_1/rs/geolocate/address'
const REGION = 'Краснодарский'

type DadataAddress = {
	street_with_type: string | null
	house_type: string | null
	house: string | null
	block_type: string | null
	block: string | null
}
type DadataResponse = { suggestions: { value: string; data: DadataAddress }[] }

/** «ул Гоголя, д 61, к 2». With two `locations` DaData keeps the region and town in `value`, so it is built from the parts. */
function streetAndHouse({ value, data }: { value: string; data: DadataAddress }): string {
	if (!data.street_with_type) return value
	const parts = [data.street_with_type]
	if (data.house) parts.push(`${data.house_type ?? 'д'} ${data.house}`)
	if (data.block) parts.push(`${data.block_type ?? 'к'} ${data.block}`)
	return parts.join(', ')
}

/** Street and house suggestions (up to 5) inside one town. No API key → no suggestions. */
export async function suggestAddress(query: string, town: string): Promise<string[]> {
	const key = serverEnv().DADATA_API_KEY
	if (!key) return []

	// Delivery prices may name a town with a prefix («п. Мирный»); DaData wants the bare name.
	const name = town.replace(/^(г|п|пос|х|ст|с|д|аул)\.?\s+/i, '').trim()
	let response: Response
	try {
		response = await fetch(ENDPOINT, {
			method: 'POST',
			headers: {
				Accept: 'application/json',
				'Content-Type': 'application/json',
				Authorization: `Token ${key}`,
			},
			body: JSON.stringify({
				query,
				count: 5,
				locations: [
					{ region: REGION, city: name },
					{ region: REGION, settlement: name },
				],
				restrict_value: true,
				from_bound: { value: 'street' },
				to_bound: { value: 'house' },
			}),
			signal: AbortSignal.timeout(3000),
		})
	} catch (error) {
		console.error('DaData address suggest failed:', error)
		throw error
	}
	if (!response.ok) {
		console.error(`DaData address suggest failed: ${response.status}`)
		throw new Error(`DaData address suggest: ${response.status}`)
	}
	const { suggestions } = (await response.json()) as DadataResponse
	return [...new Set(suggestions.map(streetAndHouse))]
}

export type ReverseAddress = {
	region: string | null
	region_type: string | null
	city: string | null
	city_type: string | null
	settlement: string | null
	settlement_type: string | null
	street_with_type: string | null
	house_type: string | null
	house: string | null
	block_type: string | null
	block: string | null
}

/** The closest DaData address for a coordinate. */
export async function reverseGeocode(lat: number, lon: number): Promise<ReverseAddress | null> {
	const key = serverEnv().DADATA_API_KEY
	if (!key) throw new Error('DaData is not configured')
	let response: Response
	try {
		response = await fetch(GEOLOCATE_ENDPOINT, {
			method: 'POST',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json', Authorization: `Token ${key}` },
			body: JSON.stringify({ lat, lon, count: 1, radius_meters: 100, language: 'ru' }),
			signal: AbortSignal.timeout(3000),
		})
	} catch (error) {
		dadataDebug('provider failure', { reason: 'Network failure or provider timeout' })
		throw new GeocodeError(
			error instanceof Error && error.name === 'TimeoutError' ? 'provider_timeout' : 'provider_network'
		)
	}
	let raw: string
	try {
		raw = await response.text()
	} catch (error) {
		throw new GeocodeError(
			error instanceof Error && error.name === 'TimeoutError' ? 'provider_timeout' : 'provider_network'
		)
	}
	dadataDebug('provider response', { httpStatus: response.status, body: raw })
	if (!response.ok) {
		dadataDebug('provider failure', { reason: 'Non-success HTTP status', httpStatus: response.status })
		throw new GeocodeError('provider_http', response.status)
	}
	let body: unknown
	try {
		body = JSON.parse(raw)
	} catch {
		dadataDebug('provider failure', { reason: 'Malformed provider JSON' })
		throw new GeocodeError('provider_json', response.status)
	}
	if (!body || typeof body !== 'object' || !('suggestions' in body) || !Array.isArray(body.suggestions)) {
		dadataDebug('provider failure', { reason: 'Missing or invalid suggestions array' })
		throw new GeocodeError('provider_shape', response.status)
	}
	if (body.suggestions.length === 0) return null
	const first = body.suggestions[0] as { data?: unknown }
	if (!first || typeof first !== 'object' || !first.data || typeof first.data !== 'object') {
		dadataDebug('provider failure', { reason: 'Missing or invalid first address' })
		throw new GeocodeError('provider_shape', response.status)
	}
	const data = first.data as Record<string, unknown>
	const fields = [
		'region',
		'region_type',
		'city',
		'city_type',
		'settlement',
		'settlement_type',
		'street_with_type',
		'house_type',
		'house',
		'block_type',
		'block',
	]
	for (const field of fields)
		if (data[field] !== undefined && data[field] !== null && typeof data[field] !== 'string') {
			dadataDebug('provider failure', { reason: 'Invalid address field type', field })
			throw new GeocodeError('provider_shape', response.status)
		}
	return Object.fromEntries(fields.map((field) => [field, data[field] ?? null])) as ReverseAddress
}
