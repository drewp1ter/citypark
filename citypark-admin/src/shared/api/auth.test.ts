import { beforeEach, describe, expect, it, vi } from 'vitest'
import { clearSession, getSession, onSessionChange, setSession } from './auth'

const store = new Map<string, string>()

beforeEach(() => {
	store.clear()
	vi.stubGlobal('sessionStorage', {
		getItem: (k: string) => store.get(k) ?? null,
		setItem: (k: string, v: string) => void store.set(k, v),
		removeItem: (k: string) => void store.delete(k),
	})
})

const session = { token: 't', email: 'a@b.c', name: 'Анна' }

describe('session storage', () => {
	it('returns what was set', () => {
		setSession(session)
		expect(getSession()).toEqual(session)
	})

	it('returns null after clear', () => {
		setSession(session)
		clearSession()
		expect(getSession()).toBeNull()
	})

	it.each(['{broken', '{"token":1}', 'null'])('drops corrupted data %s', (raw) => {
		store.set('citypark-admin-token', raw)
		expect(getSession()).toBeNull()
		expect(store.size).toBe(0)
	})

	it('notifies on set and clear until unsubscribed', () => {
		const fn = vi.fn()
		const off = onSessionChange(fn)
		setSession(session)
		clearSession()
		expect(fn).toHaveBeenCalledTimes(2)
		off()
		setSession(session)
		expect(fn).toHaveBeenCalledTimes(2)
	})
})
