'use server'

import { serverEnv } from '@/shared/config'

type Result =
	| { status: 'eligible' | 'closed' | 'canceled'; reservationNumber: string; checkIn: string; checkOut: string }
	| { status: 'invalid' | 'unavailable' | 'pending' }

export async function getCancellation(token: string): Promise<Result> {
	return request(token, 'GET')
}

export async function submitCancellation(token: string): Promise<Result> {
	const result = await request(token, 'POST')
	if (result.status !== 'pending') return result
	const current = await getCancellation(token)
	return current.status === 'canceled' || current.status === 'closed' ? current : result
}

async function request(token: string, method: 'GET' | 'POST'): Promise<Result> {
	if (typeof token !== 'string' || !/^[a-f0-9-]{36}\.[A-Za-z0-9_-]{43}$/.test(token)) return { status: 'invalid' }
	try {
		const response = await fetch(serverEnv().MENU_API_URL + '/hotel/cancellation', {
			method,
			headers: { 'X-Hotel-Cancellation-Token': token },
			cache: 'no-store',
			signal: AbortSignal.timeout(17_000),
		})
		if (response.status === 404) return { status: 'invalid' }
		if (response.status === 200 || response.status === 202) return await response.json()
	} catch {
		// The request may have reached 1C even when the response was lost.
	}
	return { status: method === 'POST' ? 'pending' : 'unavailable' }
}
