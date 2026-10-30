import 'server-only'
import { cookies } from 'next/headers'
import { serverEnv } from '@/shared/config'
import { createHash } from 'node:crypto'

const productionName = '__Host-citypark_customer'
const developmentName = 'citypark_customer_dev'
const miniName = '__Host-citypark_mini'
const miniDevName = 'citypark_mini_dev'
const googleName = '__Host-citypark_google_flow'
const googleDevName = 'citypark_google_flow_dev'
type GoogleAttempt = { key: string; token: string; intent: 'login' | 'link'; expires: number }
export const guestCookieName = 'citypark_guest_checkout_scope'

function customerCookieName() {
	return process.env.NODE_ENV === 'development' && serverEnv().CUSTOMER_PUBLIC_ORIGIN.startsWith('http://localhost:')
		? developmentName
		: productionName
}

function miniCookieName() {
	return customerCookieName() === productionName ? miniName : miniDevName
}

function googleCookieName() {
	return customerCookieName() === productionName ? googleName : googleDevName
}
function stateKey(state: string) {
	return createHash('sha256').update(state).digest('base64url')
}
function readAttempts(raw?: string): GoogleAttempt[] {
	try {
		const value: unknown = JSON.parse(raw ?? '')
		if (!Array.isArray(value)) return []
		return value
			.filter(
				(item): item is GoogleAttempt =>
					typeof item?.key === 'string' &&
					typeof item?.token === 'string' &&
					(item?.intent === 'login' || item?.intent === 'link') &&
					typeof item?.expires === 'number' &&
					item.expires > Date.now()
			)
			.slice(-3)
	} catch {
		return []
	}
}

async function writeAttempts(attempts: GoogleAttempt[]) {
	const name = googleCookieName()
	;(await cookies()).set(name, attempts.length ? JSON.stringify(attempts) : '', {
		httpOnly: true,
		secure: name === googleName,
		sameSite: 'lax',
		path: '/',
		maxAge: attempts.length ? 600 : 0,
	})
}

export async function addGoogleAttempt(state: string, token: string, intent: 'login' | 'link', expiresAt: string) {
	if (!/^[A-Za-z0-9_-]{32,128}$/.test(state) || !/^[A-Za-z0-9_-]{32,128}$/.test(token))
		throw new Error('Invalid Google attempt')
	const expires = new Date(expiresAt).getTime()
	if (!Number.isFinite(expires) || expires <= Date.now() || expires > Date.now() + 600_000)
		throw new Error('Invalid Google attempt expiry')
	const current = readAttempts((await cookies()).get(googleCookieName())?.value)
	await writeAttempts(
		[...current.filter((item) => item.key !== stateKey(state)), { key: stateKey(state), token, intent, expires }].slice(
			-3
		)
	)
}

export async function consumeGoogleAttempt(state: string): Promise<GoogleAttempt | undefined> {
	const current = readAttempts((await cookies()).get(googleCookieName())?.value)
	const key = stateKey(state)
	const found = current.find((item) => item.key === key)
	if (found) await writeAttempts(current.filter((item) => item.key !== key))
	return found
}

export async function getCustomerSessionToken(): Promise<string | undefined> {
	return (await cookies()).get(customerCookieName())?.value
}

export async function getMiniSessionToken(): Promise<string | undefined> {
	return (await cookies()).get(miniCookieName())?.value
}

export async function setMiniSession(token: string, expiresAt: string): Promise<void> {
	if (!/^cs_[A-Za-z0-9_-]+$/.test(token)) throw new Error('Invalid mini session')
	const expiry = new Date(expiresAt)
	const now = Date.now()
	if (!Number.isFinite(expiry.getTime()) || expiry.getTime() <= now || expiry.getTime() > now + 30 * 86400_000)
		throw new Error('Invalid mini session expiry')
	const name = miniCookieName()
	;(await cookies()).set(name, token, {
		httpOnly: true,
		secure: name === miniName,
		sameSite: 'lax',
		path: '/',
		expires: expiry,
	})
}

export async function hasCustomerSessionCookie(): Promise<boolean> {
	return Boolean((await cookies()).get(customerCookieName()))
}

export async function setCustomerSession(token: string, expiresAt: string): Promise<void> {
	if (!/^cs_[A-Za-z0-9_-]+$/.test(token)) throw new Error('Invalid customer session')
	const expiry = new Date(expiresAt)
	const now = Date.now()
	if (!Number.isFinite(expiry.getTime()) || expiry.getTime() <= now || expiry.getTime() > now + 30 * 86400_000)
		throw new Error('Invalid customer session expiry')
	const name = customerCookieName()
	;(await cookies()).set(name, token, {
		httpOnly: true,
		secure: name === productionName,
		sameSite: 'lax',
		path: '/',
		expires: expiry,
	})
}

export async function clearCustomerSession(): Promise<void> {
	const jar = await cookies()
	for (const name of [productionName, developmentName])
		jar.set(name, '', { httpOnly: true, secure: name === productionName, sameSite: 'lax', path: '/', maxAge: 0 })
}

export async function getGuestCheckoutCookie(): Promise<string | undefined> {
	return (await cookies()).get(guestCookieName)?.value
}

export async function saveGuestCheckoutCookie(header: string | null): Promise<void> {
	if (!header) return
	const match = /^citypark_guest_checkout=([A-Za-z0-9_.-]+)(?:;|$)/.exec(header)
	if (!match) return
	const name = customerCookieName()
	;(await cookies()).set(guestCookieName, match[1], {
		httpOnly: true,
		secure: name === productionName,
		sameSite: 'lax',
		path: '/',
		maxAge: 86400,
	})
}
