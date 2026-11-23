import 'server-only'
import { redirect } from 'next/navigation'
import type { OAuthProvider } from '@/shared/customer'
import { addOAuthAttempt, consumeOAuthAttempt, customerRequest, getCustomerSessionToken, setCustomerSession } from '@/shared/customer'

type OAuthStart = { authorizationUrl: string; flowToken: string; expiresAt: string }
type OAuthComplete = { token?: string; expiresAt?: string; linked?: boolean }

export async function startOAuth(provider: OAuthProvider, intent: 'login' | 'link') {
	if (provider === 'yandex' && !yandexEnabled()) redirect('/account/sign-in?yandex=failed')
	const session = intent === 'link' ? await getCustomerSessionToken() : undefined
	if (intent === 'link' && !session) redirect(`/account/sign-in?${provider}=expired`)
	let destination: string
	try {
		const result = await customerRequest<OAuthStart>(`/auth/customer/${provider}/start`, {
			method: 'POST', headers: session ? { Authorization: `Bearer ${session}` } : undefined,
			body: JSON.stringify({ intent }),
		})
		const expected = provider === 'yandex'
			? { origin: 'https://oauth.yandex.ru', path: '/authorize', scope: 'login:info' }
			: { origin: 'https://accounts.google.com', path: '/o/oauth2/v2/auth', scope: 'email openid profile' }
		const url = new URL(result.authorizationUrl)
		const state = url.searchParams.get('state') ?? ''
		const scopes = (url.searchParams.get('scope') ?? '').split(' ').filter(Boolean).sort().join(' ')
		if (url.protocol !== 'https:' || url.origin !== expected.origin || url.pathname !== expected.path || !/^[A-Za-z0-9_-]{32,128}$/.test(state) || scopes !== expected.scope || url.toString().includes(result.flowToken)) throw new Error('Invalid OAuth authorization URL')
		await addOAuthAttempt(provider, state, result.flowToken, intent, result.expiresAt)
		destination = url.toString()
	} catch { redirect(`/account/sign-in?${provider}=failed`) }
	redirect(destination)
}

// Relative locations keep the browser on the public origin behind a reverse proxy.
function localRedirect(provider: OAuthProvider, code: 'cancelled' | 'failed' | 'expired', linked = false): Response {
	const path = linked ? `/account?linked=${provider}` : `/account/sign-in?${provider}=${code}`
	return new Response(null, { status: 303, headers: { Location: path, 'Cache-Control': 'private, no-store', 'Referrer-Policy': 'no-referrer' } })
}

export async function completeOAuthCallback(provider: OAuthProvider, request: Request): Promise<Response> {
	const url = new URL(request.url)
	const state = url.searchParams.get('state') ?? ''
	const code = url.searchParams.get('code')
	const error = url.searchParams.get('error')
	if (!/^[A-Za-z0-9_-]{32,128}$/.test(state) || url.searchParams.getAll('state').length !== 1 || url.searchParams.getAll('code').length > 1 || url.searchParams.getAll('error').length > 1 || Boolean(code) === Boolean(error) || (code !== null && code.length > 4096) || (error !== null && error !== 'access_denied')) return localRedirect(provider, 'failed')
	const proof = await consumeOAuthAttempt(provider, state)
	if (!proof) return localRedirect(provider, 'failed')
	const session = proof.intent === 'link' ? await getCustomerSessionToken() : undefined
	if (proof.intent === 'link' && !session) return localRedirect(provider, 'expired')
	try {
		const result = await customerRequest<OAuthComplete>(`/auth/customer/${provider}/complete`, {
			method: 'POST', headers: session ? { Authorization: `Bearer ${session}` } : undefined,
			body: JSON.stringify({ state, flowToken: proof.token, ...(code ? { code } : { error }) }),
		})
		if (proof.intent === 'link') return result.linked === true ? localRedirect(provider, 'failed', true) : localRedirect(provider, 'failed')
		if (!result.token || !result.expiresAt) return localRedirect(provider, 'failed')
		await setCustomerSession(result.token, result.expiresAt)
		return new Response(null, { status: 303, headers: { Location: '/menu', 'Cache-Control': 'private, no-store', 'Referrer-Policy': 'no-referrer' } })
	} catch { return localRedirect(provider, error === 'access_denied' ? 'cancelled' : 'failed') }
}

export function yandexEnabled() { return process.env.YANDEX_SIGN_IN_ENABLED === 'true' }
