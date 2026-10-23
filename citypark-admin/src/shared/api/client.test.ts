import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getSession, setSession } from './auth'
import { api, unwrap } from './client'
import { ApiError } from './errors'

const store = new Map<string, string>()
const assign = vi.fn()
const fetchMock = vi.fn()

function json(status: number, body: unknown) {
	return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

beforeEach(() => {
	store.clear()
	vi.stubGlobal('sessionStorage', {
		getItem: (k: string) => store.get(k) ?? null,
		setItem: (k: string, v: string) => void store.set(k, v),
		removeItem: (k: string) => void store.delete(k),
	})
	vi.stubGlobal('window', { location: { pathname: '/orders', search: '?page=2', assign } })
	vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
	vi.unstubAllGlobals()
	vi.resetAllMocks()
})

describe('api client', () => {
	it('sends the Bearer token when signed in', async () => {
		setSession({ token: 'abc', email: 'a@b.c', name: 'A' })
		fetchMock.mockResolvedValue(json(200, {}))
		await api.GET('/user')
		const request = fetchMock.mock.calls[0]?.[0] as Request
		expect(request.headers.get('Authorization')).toBe('Bearer abc')
	})

	it('sends no header without a session', async () => {
		fetchMock.mockResolvedValue(json(200, {}))
		await api.GET('/user')
		const request = fetchMock.mock.calls[0]?.[0] as Request
		expect(request.headers.has('Authorization')).toBe(false)
	})

	it('clears the session and redirects on a 401', async () => {
		setSession({ token: 'abc', email: 'a@b.c', name: 'A' })
		fetchMock.mockResolvedValue(json(401, { statusCode: 401, message: 'Требуется авторизация' }))
		await api.GET('/user')
		expect(getSession()).toBeNull()
		expect(assign).toHaveBeenCalledWith('/login?next=' + encodeURIComponent('/orders?page=2'))
	})

	it('does not redirect on a 401 from sign-in', async () => {
		fetchMock.mockResolvedValue(json(401, { statusCode: 401, message: 'Неверный пароль' }))
		const error = await unwrap(api.POST('/auth/sign-in', { body: { email: 'a@b.c', password: 'x' } })).catch(
			(e: unknown) => e
		)
		expect(assign).not.toHaveBeenCalled()
		expect(error).toBeInstanceOf(ApiError)
		expect((error as ApiError).message).toBe('Неверный пароль')
	})
})

describe('unwrap', () => {
	it('throws ApiError with fields on a 400', async () => {
		fetchMock.mockResolvedValue(json(400, { statusCode: 400, message: 'Ошибка валидации', errors: { email: 'bad' } }))
		const error = await unwrap(api.POST('/auth/sign-in', { body: { email: '', password: '' } })).catch(
			(e: unknown) => e
		)
		expect(error).toBeInstanceOf(ApiError)
		expect((error as ApiError).status).toBe(400)
		expect((error as ApiError).fields).toEqual({ email: 'bad' })
	})

	it('turns a network failure into status 0', async () => {
		fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))
		const error = await unwrap(api.GET('/user')).catch((e: unknown) => e)
		expect((error as ApiError).status).toBe(0)
	})

	it('returns undefined for a 204', async () => {
		fetchMock.mockResolvedValue(new Response(null, { status: 204 }))
		await expect(unwrap(api.GET('/user'))).resolves.toBeUndefined()
	})

	it('returns the data on success', async () => {
		fetchMock.mockResolvedValue(json(200, { id: 1 }))
		await expect(unwrap(api.GET('/user'))).resolves.toEqual({ id: 1 })
	})
})
