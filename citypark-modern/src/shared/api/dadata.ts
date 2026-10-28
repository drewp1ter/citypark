import { serverEnv } from '@/shared/config'

const ENDPOINT = 'https://suggestions.dadata.ru/suggestions/api/4_1/rs/suggest/address'
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
