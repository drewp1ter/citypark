import { z } from 'zod'

const schema = z.object({
	MENU_API_URL: z.url('MENU_API_URL must be a URL, e.g. http://localhost:3000/api/v1'),
	UPLOADS_ORIGIN: z.preprocess((value) => value || undefined, z.url().optional()),
	YANDEX_MAPS_API_KEY: z.preprocess((value) => value || undefined, z.string().optional()),
	DADATA_API_KEY: z.preprocess((value) => value || undefined, z.string().optional()),
})

/** Server-side settings. Parsed on each call, not at import time, so a build without them fails only where they are read. */
export function serverEnv(): {
	MENU_API_URL: string
	UPLOADS_ORIGIN?: string
	YANDEX_MAPS_API_KEY?: string
	DADATA_API_KEY?: string
} {
	const { MENU_API_URL, UPLOADS_ORIGIN, YANDEX_MAPS_API_KEY, DADATA_API_KEY } = schema.parse({
		MENU_API_URL: process.env.MENU_API_URL,
		UPLOADS_ORIGIN: process.env.UPLOADS_ORIGIN,
		YANDEX_MAPS_API_KEY: process.env.YANDEX_MAPS_API_KEY,
		DADATA_API_KEY: process.env.DADATA_API_KEY,
	})
	return {
		MENU_API_URL: MENU_API_URL.replace(/\/+$/, ''),
		UPLOADS_ORIGIN: UPLOADS_ORIGIN?.replace(/\/+$/, ''),
		YANDEX_MAPS_API_KEY,
		DADATA_API_KEY,
	}
}
