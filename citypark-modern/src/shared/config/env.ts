import { z } from 'zod'

const schema = z.object({
	MENU_API_URL: z.url('MENU_API_URL must be a URL, e.g. http://localhost:3000/api/v1'),
})

/** Server-side settings. Parsed on each call, not at import time, so a build without them fails only where they are read. */
export function serverEnv(): { MENU_API_URL: string } {
	const { MENU_API_URL } = schema.parse({ MENU_API_URL: process.env.MENU_API_URL })
	return { MENU_API_URL: MENU_API_URL.replace(/\/+$/, '') }
}
