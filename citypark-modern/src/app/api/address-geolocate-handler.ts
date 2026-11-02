import type { AutofillReason, AutofillDetails } from '@/shared/monitoring'

export type GeolocateAddress = {
	region: string | null
	region_type: string | null
	city: string | null
	settlement: string | null
	street_with_type: string | null
	house_type: string | null
	house: string | null
	block_type: string | null
	block: string | null
}
export type GeolocateTown = { name: string; price: number; freeFrom: number }
export type GeolocateDependencies = {
	report?: (reason: AutofillReason, details: AutofillDetails) => void
	key: () => string | undefined
	geocode: (lat: number, lon: number) => Promise<GeolocateAddress | null>
	towns: () => Promise<readonly GeolocateTown[]>
	match: (locality: string | null | undefined, towns: readonly GeolocateTown[]) => GeolocateTown | undefined
	normalize?: (locality: string) => string
	debug?: (event: string, details: unknown) => void
}

const headers = { 'Cache-Control': 'no-store' }
const json = (body: object, status = 200) => Response.json(body, { status, headers })

function street(data: GeolocateAddress): string {
	if (!data.street_with_type?.trim()) return ''
	const parts = [data.street_with_type.trim()]
	if (data.house?.trim()) parts.push(`${data.house_type?.trim() || 'д'} ${data.house.trim()}`)
	if (data.block?.trim()) parts.push(`${data.block_type?.trim() || 'к'} ${data.block.trim()}`)
	return parts.join(', ')
}

export async function postAddressGeolocate(request: Request, deps: GeolocateDependencies): Promise<Response> {
	const started = Date.now()
	const report = (reason: AutofillReason, stage: AutofillDetails['stage'], details: Partial<AutofillDetails> = {}) => {
		deps.report?.(reason, { ...details, stage, elapsed_ms: Date.now() - started })
	}
	const invalid = () => {
		report('invalid', 'validation')
		return json({ status: 'invalid' }, 400)
	}
	let input: unknown
	try {
		input = await request.json()
	} catch {
		return invalid()
	}
	if (!input || typeof input !== 'object') return invalid()
	const { lat, lon } = input as Record<string, unknown>
	if (
		typeof lat !== 'number' ||
		!Number.isFinite(lat) ||
		lat < -90 ||
		lat > 90 ||
		typeof lon !== 'number' ||
		!Number.isFinite(lon) ||
		lon < -180 ||
		lon > 180
	)
		return invalid()
	let key: string | undefined
	try {
		key = deps.key()
	} catch {
		deps.debug?.('selection skipped', { reason: 'Server configuration unavailable' })
		report('configuration', 'configuration')
		return json({ status: 'unavailable' }, 503)
	}
	if (!key) {
		deps.debug?.('selection skipped', { reason: 'Missing DADATA_API_KEY' })
		report('configuration', 'configuration')
		return json({ status: 'unavailable' }, 503)
	}
	let stage = 'provider lookup'
	try {
		const address = await deps.geocode(lat, lon)
		if (!address) {
			report('empty_results', 'provider')
			deps.debug?.('selection rejected', { reason: 'Empty provider results' })
			return json({ status: 'unmatched' })
		}
		deps.debug?.('provider locality', {
			region: address.region,
			regionType: address.region_type,
			city: address.city,
			settlement: address.settlement,
		})
		if (address.region !== 'Краснодарский' || address.region_type !== 'край') {
			report('unsupported_region', 'matching')
			deps.debug?.('selection rejected', { reason: 'Unsupported region' })
			return json({ status: 'unmatched' })
		}
		const locality = address.settlement?.trim() ? address.settlement : address.city
		stage = 'delivery-list lookup'
		const towns = await deps.towns()
		stage = 'town matching'
		const town = deps.match(locality, towns)
		const normalize = deps.normalize
		if (normalize && deps.debug) {
			const normalized = normalize(locality || '')
			const deliveryTowns = towns.map(({ name }) => ({ name, normalized: normalize(name) }))
			const candidates = deliveryTowns.filter((item) => item.normalized === normalized)
			deps.debug('town matching', {
				locality,
				source: address.settlement?.trim() ? 'settlement' : 'city',
				normalized,
				deliveryTowns,
				candidates,
				reason: town ? 'Matched' : candidates.length > 1 ? 'Multiple town matches' : 'No town match',
			})
		}
		if (!town) {
			const candidateCount = deps.normalize
				? towns.filter(({ name }) => deps.normalize!(name) === deps.normalize!(locality || '')).length
				: 0
			report(candidateCount > 1 ? 'ambiguous_town' : 'no_town', 'matching', {
				town_count: towns.length,
				candidate_count: candidateCount,
				locality_source: address.settlement?.trim() ? 'settlement' : 'city',
			})
			return json({ status: 'unmatched' })
		}
		if (!street(address)) report('partial', 'matching')
		deps.debug?.('selection accepted', { town: town.name, street: street(address) })
		return json({ status: 'matched', town: town.name, street: street(address) })
	} catch (error) {
		const failure = error as { reason?: AutofillReason; httpStatus?: number } | null
		const reasons = ['provider_network', 'provider_timeout', 'provider_http', 'provider_json', 'provider_shape']
		report(
			stage === 'delivery-list lookup'
				? 'delivery_list'
				: stage === 'town matching'
					? 'town_matching'
					: failure?.reason && reasons.includes(failure.reason)
						? failure.reason
						: 'provider_network',
			stage === 'delivery-list lookup' ? 'delivery_list' : stage === 'town matching' ? 'matching' : 'provider',
			{ http_status: failure?.httpStatus }
		)
		deps.debug?.('selection failed', { reason: `${stage} failed` })
		return json({ status: 'unavailable' }, 502)
	}
}
