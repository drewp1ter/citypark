import 'server-only'
import { cookies } from 'next/headers'
import { serverEnv } from '@/shared/config'

const productionName = '__Host-citypark_customer'
const developmentName = 'citypark_customer_dev'
export const guestCookieName = 'citypark_guest_checkout_scope'

function customerCookieName() {
	return process.env.NODE_ENV === 'development' && serverEnv().CUSTOMER_PUBLIC_ORIGIN.startsWith('http://localhost:')
		? developmentName
		: productionName
}

export async function getCustomerSessionToken(): Promise<string | undefined> {
	return (await cookies()).get(customerCookieName())?.value
}

export async function hasCustomerSessionCookie(): Promise<boolean> {
	return Boolean((await cookies()).get(customerCookieName()))
}

export async function setCustomerSession(token: string, expiresAt: string): Promise<void> {
	if (!/^cs_[A-Za-z0-9_-]+$/.test(token)) throw new Error('Invalid customer session')
	const expiry = new Date(expiresAt)
	const now = Date.now()
	if (!Number.isFinite(expiry.getTime()) || expiry.getTime() <= now || expiry.getTime() > now + 30 * 86400_000) throw new Error('Invalid customer session expiry')
	const name = customerCookieName()
	;(await cookies()).set(name, token, { httpOnly: true, secure: name === productionName, sameSite: 'lax', path: '/', expires: expiry })
}

export async function clearCustomerSession(): Promise<void> {
	const jar = await cookies()
	for (const name of [productionName, developmentName]) jar.set(name, '', { httpOnly: true, secure: name === productionName, sameSite: 'lax', path: '/', maxAge: 0 })
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
		httpOnly: true, secure: name === productionName, sameSite: 'lax', path: '/', maxAge: 86400,
	})
}
