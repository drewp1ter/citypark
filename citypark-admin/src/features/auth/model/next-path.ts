const DEFAULT_PATH = '/orders'

/** A safe in-app path to return to after sign-in; anything else gives `/orders`. */
export function safeNextPath(next: string | undefined): string {
	if (next === undefined || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) {
		return DEFAULT_PATH
	}
	const pathname = next.split(/[?#]/)[0]
	if (pathname === '/login') return DEFAULT_PATH
	return next
}
