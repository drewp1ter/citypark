import 'server-only'
import { cookies } from 'next/headers'
import { createHmac, randomBytes, timingSafeEqual } from 'node:crypto'
import { serverEnv } from '@/shared/config'

function settings() {
	const secret = process.env.HOTEL_BOOKING_COOKIE_SECRET
	if (!secret || Buffer.byteLength(secret) < 32) throw new Error('Missing booking cookie signing secret')
	const secure = new URL(serverEnv().CUSTOMER_PUBLIC_ORIGIN).protocol === 'https:'
	return { secret, secure, name: secure ? '__Host-citypark_booking' : 'citypark_booking_dev' }
}
export async function getBookingContext(create = false): Promise<string | undefined> {
	const { secret, name, secure } = settings()
	const jar = await cookies()
	const raw = jar.get(name)?.value
	if (raw) {
		const [token, signature, extra] = raw.split('.')
		if (!extra && /^[A-Za-z0-9_-]{43}$/.test(token ?? '') && /^[A-Za-z0-9_-]{43}$/.test(signature ?? '')) {
			const expected = createHmac('sha256', secret).update(token).digest()
			const actual = Buffer.from(signature, 'base64url')
			if (actual.length === expected.length && timingSafeEqual(actual, expected)) return token
		}
	}
	if (!create) return undefined
	const token = randomBytes(32).toString('base64url')
	const signature = createHmac('sha256', secret).update(token).digest('base64url')
	jar.set(name, token + '.' + signature, { httpOnly: true, secure, sameSite: 'lax', path: '/', maxAge: 365 * 86400 })
	return token
}
