import { getDeliveryTowns } from '@/entities/dish'
import { suggestAddress } from '@/shared/api'

const MIN_QUERY = 3
const MAX_QUERY = 120

/** Address suggestions for the checkout form. Only delivery towns are accepted, so the DaData key is not exposed as an open proxy. */
export async function GET(request: Request) {
	const params = new URL(request.url).searchParams
	const query = (params.get('q') ?? '').trim()
	const town = params.get('town') ?? ''

	const towns = await getDeliveryTowns()
	if (!towns.some((item) => item.name === town)) return Response.json([], { status: 400 })
	if (query.length < MIN_QUERY || query.length > MAX_QUERY) return Response.json([])

	try {
		const suggestions = await suggestAddress(query, town)
		return Response.json(suggestions, { headers: { 'Cache-Control': 'private, max-age=300' } })
	} catch {
		return Response.json([], { status: 502 })
	}
}
