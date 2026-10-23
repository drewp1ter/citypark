import { z } from 'zod'

const schema = z.object({
	MENU_API_URL: z.url('MENU_API_URL must be a URL, e.g. http://localhost:3000/api/v1'),
	UPLOADS_ORIGIN: z.preprocess((value) => value || undefined, z.url().optional()),
})

/** Server-side settings. Parsed on each call, not at import time, so a build without them fails only where they are read. */
export function serverEnv(): { MENU_API_URL: string; UPLOADS_ORIGIN?: string } {
	const { MENU_API_URL, UPLOADS_ORIGIN } = schema.parse({
		MENU_API_URL: process.env.MENU_API_URL,
		UPLOADS_ORIGIN: process.env.UPLOADS_ORIGIN,
	})
	return { MENU_API_URL: MENU_API_URL.replace(/\/+$/, ''), UPLOADS_ORIGIN: UPLOADS_ORIGIN?.replace(/\/+$/, '') }
}
