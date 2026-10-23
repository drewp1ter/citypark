const KEY = 'citypark-admin-token'

export type Session = { token: string; email: string; name: string }

const listeners = new Set<() => void>()

function notify() {
	listeners.forEach((fn) => fn())
}

/** The saved session, or null. Corrupted data is removed. */
export function getSession(): Session | null {
	const raw = sessionStorage.getItem(KEY)
	if (raw === null) return null
	try {
		const value: unknown = JSON.parse(raw)
		if (
			typeof value === 'object' &&
			value !== null &&
			'token' in value &&
			'email' in value &&
			'name' in value &&
			typeof value.token === 'string' &&
			typeof value.email === 'string' &&
			typeof value.name === 'string'
		) {
			return { token: value.token, email: value.email, name: value.name }
		}
	} catch {
		// fall through to removal
	}
	sessionStorage.removeItem(KEY)
	return null
}

export function setSession(s: Session): void {
	sessionStorage.setItem(KEY, JSON.stringify(s))
	notify()
}

export function clearSession(): void {
	sessionStorage.removeItem(KEY)
	notify()
}

/** Subscribe to sign-in/sign-out in this tab (for React): returns unsubscribe. */
export function onSessionChange(fn: () => void): () => void {
	listeners.add(fn)
	return () => {
		listeners.delete(fn)
	}
}
