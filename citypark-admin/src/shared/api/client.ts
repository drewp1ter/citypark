import createClient, { type Middleware } from 'openapi-fetch'
import { clearSession, getSession } from './auth'
import { ApiError, toApiError } from './errors'
import type { paths } from './schema'

const baseUrl = import.meta.env.VITE_API_URL ?? 'http://localhost:3000/api/v1'

function isSignIn(url: string): boolean {
	return new URL(url).pathname.endsWith('/auth/sign-in')
}

function redirectToLogin() {
	clearSession()
	const next = window.location.pathname + window.location.search
	window.location.assign('/login?next=' + encodeURIComponent(next))
}

const auth: Middleware = {
	onRequest({ request }) {
		const session = getSession()
		if (session) request.headers.set('Authorization', `Bearer ${session.token}`)
		return request
	},
	onResponse({ request, response }) {
		if (response.status === 401 && !isSignIn(request.url)) redirectToLogin()
		return response
	},
}

export const api = createClient<paths>({
	baseUrl,
	// Looked up on every call so tests (and polyfills) can replace the global fetch.
	fetch: (request) => globalThis.fetch(request),
})
api.use(auth)

/** Unwraps an openapi-fetch result: returns data, or throws ApiError. A 204 gives undefined. */
export async function unwrap<T>(p: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
	let result
	try {
		result = await p
	} catch (e) {
		if (e instanceof ApiError) throw e
		throw toApiError(0, undefined)
	}
	if (result.error !== undefined || !result.response.ok) {
		throw toApiError(result.response.status, result.error)
	}
	return result.data as T
}

/** Uploads one file as multipart field `file` (task 08). Same auth header and error handling as `api`. */
export async function apiUpload(path: string, file: File): Promise<{ url: string }> {
	const headers = new Headers()
	const session = getSession()
	if (session) headers.set('Authorization', `Bearer ${session.token}`)
	const form = new FormData()
	form.append('file', file)
	let response: Response
	try {
		response = await fetch(baseUrl + path, { method: 'POST', headers, body: form })
	} catch {
		throw toApiError(0, undefined)
	}
	const body: unknown = await response.json().catch(() => undefined)
	if (!response.ok) {
		if (response.status === 401) redirectToLogin()
		throw toApiError(response.status, body)
	}
	return body as { url: string }
}
