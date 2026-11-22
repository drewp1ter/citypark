// Isolated development acceptance only. Preserves room content and restores its rate.
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import env from '@next/env'

env.loadEnvConfig(process.cwd())
const api = process.env.MENU_API_URL
let token
const editable = [
	'name',
	'hotelName',
	'numbers',
	'area',
	'areaDetailed',
	'capacity',
	'summary',
	'description',
	'pricePerNight',
	'photo',
	'gallery',
	'hotelNote',
]
async function request(path, body) {
	const response = await fetch(api + path, {
		method: body === undefined ? 'GET' : 'PUT',
		headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: 'Bearer ' + token } : {}) },
		body: body === undefined ? undefined : JSON.stringify(body),
		redirect: 'error',
		signal: AbortSignal.timeout(15000),
	})
	assert.ok(response.ok, `Development room operation failed: HTTP ${response.status}`)
	return response.json()
}
async function authenticate() {
	if (token) return
	const values = {}
	for (const line of (await readFile('../citypark-go/.env', 'utf8')).split(/\r?\n/)) {
		if (line.trim().startsWith('#') || !line.includes('=')) continue
		const at = line.indexOf('=')
		values[line.slice(0, at).trim()] = line
			.slice(at + 1)
			.trim()
			.replace(/^(["'])(.*)\1$/, '$2')
	}
	assert.ok(values.ADMIN_EMAIL && values.ADMIN_PASSWORD, 'Development administrator credentials unavailable')
	const response = await fetch(api + '/auth/sign-in', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ email: values.ADMIN_EMAIL, password: values.ADMIN_PASSWORD }),
		redirect: 'error',
		signal: AbortSignal.timeout(10000),
	})
	assert.equal(response.status, 200, 'Development administrator authentication failed')
	const result = await response.json()
	assert.equal(result.role, 'admin')
	token = result.token
}
export async function changeRoomRate(roomTypeId) {
	assert.equal(process.env.HOTEL_BOOKING_QA_APPROVED, '1')
	await authenticate()
	const original = (await request('/rooms')).find((room) => room.id === roomTypeId)
	assert.ok(original)
	assert.ok(
		!original.photo.position || /^\d+(?:\.\d+)?% \d+(?:\.\d+)?%$/.test(original.photo.position),
		'Legacy photo position cannot be round-tripped unchanged; no price edit sent'
	)
	const payload = Object.fromEntries(editable.map((key) => [key, original[key]]))
	const changedPrice = original.pricePerNight + 1
	await request('/rooms/' + roomTypeId, { ...payload, pricePerNight: changedPrice })
	console.log('Changed isolated room rate by 1 RUB for stale-quote acceptance; restoration is registered.')
	return async () => {
		const current = (await request('/rooms')).find((room) => room.id === roomTypeId)
		assert.equal(
			current.pricePerNight,
			changedPrice,
			'Room rate changed concurrently; manual review required before restoration'
		)
		const restored = await request('/rooms/' + roomTypeId, {
			...Object.fromEntries(editable.map((key) => [key, current[key]])),
			pricePerNight: original.pricePerNight,
		})
		assert.equal(restored.pricePerNight, original.pricePerNight)
		console.log('Restored original isolated room rate.')
	}
}
