import 'server-only'
import { redirect } from 'next/navigation'
import { addGoogleAttempt, consumeGoogleAttempt, customerRequest, getCustomerSessionToken, setCustomerSession } from '@/shared/customer'

type GoogleStart = { authorizationUrl: string; flowToken: string; expiresAt: string }
type GoogleComplete = { token?: string; expiresAt?: string; linked?: boolean }

async function start(intent: 'login' | 'link') {
	const session = intent === 'link' ? await getCustomerSessionToken() : undefined
	if (intent === 'link' && !session) redirect('/account/sign-in?google=expired')
	let destination: string
	try {
		const result = await customerRequest<GoogleStart>('/auth/customer/google/start', {
			method: 'POST', headers: session ? { Authorization: `Bearer ${session}` } : undefined,
			body: JSON.stringify({ intent }),
		})
		const url = new URL(result.authorizationUrl)
		const state = url.searchParams.get('state') ?? ''
		const scopes = (url.searchParams.get('scope') ?? '').split(' ').filter(Boolean).sort().join(' ')
		if (url.protocol !== 'https:' || url.hostname !== 'accounts.google.com' || url.pathname !== '/o/oauth2/v2/auth' || !/^[A-Za-z0-9_-]{32,128}$/.test(state) || scopes !== 'email openid profile' || url.toString().includes(result.flowToken)) throw new Error('Invalid Google authorization URL')
		await addGoogleAttempt(state, result.flowToken, intent, result.expiresAt)
		destination = url.toString()
	} catch { redirect('/account/sign-in?google=failed') }
	redirect(destination)
}

export async function startGoogleSignIn(): Promise<void> { await start('login') }
export async function startGoogleLink(): Promise<void> { await start('link') }

function localRedirect(request: Request, code: 'cancelled' | 'failed' | 'expired', linked = false): Response {
	const path = linked ? '/account?linked=google' : `/account/sign-in?google=${code}`
	return new Response(null, { status: 303, headers: { Location: new URL(path, request.url).toString(), 'Cache-Control': 'private, no-store', 'Referrer-Policy': 'no-referrer' } })
}

export async function completeGoogleCallback(request: Request): Promise<Response> {
	const url = new URL(request.url)
	const state = url.searchParams.get('state') ?? ''
	const code = url.searchParams.get('code')
	const error = url.searchParams.get('error')
	if (!/^[A-Za-z0-9_-]{32,128}$/.test(state) || url.searchParams.getAll('state').length !== 1 || url.searchParams.getAll('code').length > 1 || url.searchParams.getAll('error').length > 1 || Boolean(code) === Boolean(error) || (code !== null && code.length > 4096) || (error !== null && error !== 'access_denied')) return localRedirect(request, 'failed')
	const proof = await consumeGoogleAttempt(state)
	if (!proof) return localRedirect(request, 'failed')
	const session = proof.intent === 'link' ? await getCustomerSessionToken() : undefined
	if (proof.intent === 'link' && !session) return localRedirect(request, 'expired')
	try {
		const result = await customerRequest<GoogleComplete>('/auth/customer/google/complete', {
			method: 'POST', headers: session ? { Authorization: `Bearer ${session}` } : undefined,
			body: JSON.stringify({ state, flowToken: proof.token, ...(code ? { code } : { error }) }),
		})
		if (proof.intent === 'link') return result.linked === true ? localRedirect(request, 'failed', true) : localRedirect(request, 'failed')
		if (!result.token || !result.expiresAt) return localRedirect(request, 'failed')
		await setCustomerSession(result.token, result.expiresAt)
		return new Response(null, { status: 303, headers: { Location: new URL('/menu', request.url).toString(), 'Cache-Control': 'private, no-store', 'Referrer-Policy': 'no-referrer' } })
	} catch { return localRedirect(request, error === 'access_denied' ? 'cancelled' : 'failed') }
}
