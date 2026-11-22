// Isolated protocol, signing and recovery tests. No real HTTP calls or reservations.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import * as crypto from 'node:crypto'
import { z } from 'zod'
function load(file, modules, globals = {}) {
	const exports = {}
	const source = ts.transpileModule(readFileSync(file, 'utf8'), {
		compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
	}).outputText
	vm.runInNewContext(source, {
		exports,
		require: (name) => {
			assert.ok(name in modules, name)
			return modules[name]
		},
		Buffer,
		URL,
		Date,
		TextEncoder,
		AbortSignal,
		...globals,
	})
	return exports
}
const lib = { todayInMoscow: () => '2026-11-22' }
const model = load('src/entities/room/model/booking.ts', { zod: { z }, '@/shared/lib': lib })
const schema = load('src/pages/booking/model/schema.ts', { zod: { z }, '@/entities/room/index.client': model })
const stay = { roomTypeId: 'single-balcony', checkIn: '2026-12-01', checkOut: '2026-12-03', guests: 1 }
const key = crypto.randomUUID(),
	quoteId = crypto.randomUUID()
const quote = {
	...stay,
	quoteId,
	expiresAt: '2026-11-22T12:10:00Z',
	available: true,
	price: { currency: 'RUB', nights: 2, pricePerNight: 4000, total: 8000 },
}
const guest = { name: 'Иван Иванов', email: 'Ivan@example.com', phone: '+79180000000' }
const request = { idempotencyKey: key, quoteId, guest }
const reservation = {
	...stay,
	reservationId: crypto.randomUUID(),
	status: 'confirmed',
	price: quote.price,
	confirmationEmailStatus: 'pending',
}
assert.equal(model.bookingStaySchema.safeParse(stay).success, true)
assert.equal(
	model.bookingQuoteSchema.safeParse({ ...quote, expiresAt: '2026-11-22T22:12:06.31696893+03:00' }).success,
	true
)
assert.equal(model.bookingQuoteSchema.safeParse({ ...quote, expiresAt: '2026-11-22T22:12:06' }).success, false)
for (const patch of [
	{ checkIn: '2026-11-22' },
	{ checkIn: '2027-02-29' },
	{ checkOut: '2026-12-01' },
	{ checkOut: '4000-01-01' },
	{ guests: 2 },
	{ guests: 1.5 },
	{ extra: true },
])
	assert.equal(model.bookingStaySchema.safeParse({ ...stay, ...patch }).success, false)
for (const dates of [
	['2028-02-29', '2028-03-01'],
	['2027-01-01', '2027-12-31'],
	['3999-12-30', '3999-12-31'],
])
	assert.equal(model.bookingStaySchema.safeParse({ ...stay, checkIn: dates[0], checkOut: dates[1] }).success, true)
assert.equal(
	schema.guestSchema.parse({ name: ' Иван ', email: 'Ivan@EXAMPLE.COM ', phone: '8 (918) 000-00-00' }).phone,
	guest.phone
)
assert.equal(schema.guestSchema.parse({ ...guest, email: 'Ivan@EXAMPLE.COM ' }).email, guest.email)
for (const patch of [
	{ name: '' },
	{ name: 'x'.repeat(151) },
	{ email: 'invalid' },
	{ phone: '+123' },
	{ name: 'a\nb' },
	{ email: 'a\r@example.com' },
	{ passport: 'x' },
])
	assert.equal(schema.guestSchema.safeParse({ ...guest, ...patch }).success, false)
assert.equal(
	model.bookingStaySchema.safeParse({ ...stay, guests: 99 }).error.issues[0].message,
	'Укажите число гостей от 1 до 4'
)
console.log('PASS dates, capacity, exact occupancy, contacts, normalization and unknown fields')

const jar = new Map()
let cookieOptions
const context = load(
	'src/pages/booking/api/context.ts',
	{
		'server-only': {},
		'node:crypto': crypto,
		'@/shared/config': { serverEnv: () => ({ CUSTOMER_PUBLIC_ORIGIN: 'https://hotel.example' }) },
		'next/headers': {
			cookies: async () => ({
				get: (name) => (jar.has(name) ? { value: jar.get(name) } : undefined),
				set: (name, value, options) => {
					jar.set(name, value)
					cookieOptions = options
				},
			}),
		},
	},
	{ process: { env: { HOTEL_BOOKING_COOKIE_SECRET: 's'.repeat(48) } } }
)
assert.equal(await context.getBookingContext(), undefined)
const token = await context.getBookingContext(true)
assert.equal(Buffer.from(token, 'base64url').length, 32)
assert.equal(await context.getBookingContext(), token)
assert.equal(cookieOptions.httpOnly, true)
assert.equal(cookieOptions.secure, true)
assert.equal(cookieOptions.sameSite, 'lax')
const original = jar.get('__Host-citypark_booking')
jar.set('__Host-citypark_booking', original + 'bad')
assert.equal(await context.getBookingContext(), undefined)
assert.notEqual(await context.getBookingContext(true), token)
console.log('PASS signed context persistence, tamper rejection and cookie flags')

let response, networkError, call
const api = load(
	'src/entities/room/api/booking-api.ts',
	{
		'server-only': {},
		'node:crypto': crypto,
		zod: { z },
		'../model/booking': model,
		'@/shared/config': { serverEnv: () => ({ MENU_API_URL: 'https://backend.example/api/v1' }) },
	},
	{
		fetch: async (url, init) => {
			call = { url, init }
			if (networkError) throw networkError
			return response
		},
	}
)
const json = (value, status = 200) => new Response(JSON.stringify(value), { status })
response = json(quote)
await api.fetchBookingQuote(token, stay)
assert.equal(call.init.cache, 'no-store')
assert.equal(call.init.redirect, 'error')
assert.equal(call.init.headers['X-Hotel-Booking-Context'], token)
response = json({ ...quote, guests: 2 })
await assert.rejects(api.fetchBookingQuote(token, stay))
response = json({ status: 'pending' }, 202)
assert.equal((await api.bookingRequest('/reservations', token, request)).status, 'pending')
response = json(reservation, 201)
assert.equal((await api.bookingRequest('/reservations', token, request)).reservationId, reservation.reservationId)
response = new Response('x'.repeat(65_537))
await assert.rejects(api.bookingRequest('/reservations/' + key, token), /Oversized/)
response = json({ statusCode: 503, code: 'INTERNAL_DEBUG', message: 'secret' }, 503)
await assert.rejects(api.bookingRequest('/reservations/' + key, token))
console.log('PASS private fetch, bounded response, quote identity, pending and malformed upstream handling')

let contextToken = token
const actions = load('src/pages/booking/api/actions.ts', {
	'@/entities/room': model,
	'@/entities/room/index.server': api,
	'../model/schema': schema,
	'./context': { getBookingContext: async () => contextToken },
})
networkError = Error('lost response containing private data')
let state = await actions.createReservationAction(request, quote)
assert.equal(state.status, 'pending')
assert.equal(state.idempotencyKey, key)
assert.ok(!JSON.stringify(state).includes('private data'))
networkError = undefined
for (const [code, status] of [
	['PRICE_CHANGED', 409],
	['QUOTE_EXPIRED', 409],
	['ROOM_UNAVAILABLE', 409],
	['RATE_LIMITED', 429],
	['BOOKING_UNAVAILABLE', 503],
]) {
	response = json({ statusCode: status, code, message: 'untrusted upstream text' }, status)
	state = await actions.createReservationAction(request, quote)
	assert.equal(state.status, 'error')
	assert.equal(state.code, code)
	assert.equal(state.idempotencyKey, key)
	assert.equal(state.message, api.bookingMessages[code])
}
response = json({ statusCode: 404, code: 'RESERVATION_NOT_FOUND', message: 'not found' }, 404)
state = await actions.lookupReservationAction(key)
assert.equal(state.status, 'unknown')
assert.equal(state.idempotencyKey, key)
response = json(reservation)
assert.equal((await actions.lookupReservationAction(key)).status, 'confirmed')
contextToken = undefined
assert.equal((await actions.createReservationAction(request, quote)).status, 'unknown')
assert.equal((await actions.lookupReservationAction('invalid')).status, 'error')
console.log('PASS lost-response safety, definitive errors, same-key recovery and missing-context safety')

const storage = new Map()
let url = new URL('https://hotel.example/booking?roomTypeId=single-balcony')
const attempts = load(
	'src/pages/booking/model/attempt.ts',
	{ '@/entities/room/index.client': model, './schema': schema },
	{
		sessionStorage: { setItem: (k, v) => storage.set(k, v), getItem: (k) => storage.get(k) },
		location: {
			get href() {
				return url.href
			},
		},
		history: {
			replaceState: (_state, _title, next) => {
				url = new URL(next)
			},
		},
	}
)
attempts.rememberAttempt({ request, quote })
assert.equal(url.searchParams.get('attempt'), key)
assert.equal(attempts.restoreAttempt(key).request.idempotencyKey, key)
assert.equal(attempts.restoreAttempt(key).quote.quoteId, quoteId)
assert.equal(attempts.restoreAttempt(key).values.guest.phone, guest.phone)
storage.set('citypark-hotel-attempt:' + key, '{broken')
assert.equal(attempts.restoreAttempt(key), undefined)
assert.equal(attempts.restoreAttempt(crypto.randomUUID()), undefined)
console.log('PASS pre-dispatch persistence, reload snapshot, expired-quote recovery and corrupt draft handling')
