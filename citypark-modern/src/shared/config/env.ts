import { z } from 'zod'

const schema = z.object({
	MENU_API_URL: z.url('MENU_API_URL must be a URL, e.g. http://localhost:3000/api/v1'),
	CUSTOMER_PUBLIC_ORIGIN: z.url('CUSTOMER_PUBLIC_ORIGIN must be a URL'),
	UPLOADS_ORIGIN: z.preprocess((value) => value || undefined, z.url().optional()),
	YANDEX_MAPS_API_KEY: z.preprocess((value) => value || undefined, z.string().optional()),
	DADATA_API_KEY: z.preprocess((value) => value || undefined, z.string().optional()),
	TELEGRAM_MINI_APP_URL: z.preprocess((value) => value || undefined, z.url().optional()),
})

/** Server-side settings. Parsed on each call, not at import time, so a build without them fails only where they are read. */
export function serverEnv(): {
	MENU_API_URL: string
	CUSTOMER_PUBLIC_ORIGIN: string
	UPLOADS_ORIGIN?: string
	YANDEX_MAPS_API_KEY?: string
	DADATA_API_KEY?: string
	TELEGRAM_MINI_APP_URL?: string
} {
	const { MENU_API_URL, CUSTOMER_PUBLIC_ORIGIN, UPLOADS_ORIGIN, YANDEX_MAPS_API_KEY, DADATA_API_KEY, TELEGRAM_MINI_APP_URL } = schema.parse({
		MENU_API_URL: process.env.MENU_API_URL,
		CUSTOMER_PUBLIC_ORIGIN: process.env.CUSTOMER_PUBLIC_ORIGIN ?? 'http://localhost:3001',
		UPLOADS_ORIGIN: process.env.UPLOADS_ORIGIN,
		YANDEX_MAPS_API_KEY: process.env.YANDEX_MAPS_API_KEY,
		DADATA_API_KEY: process.env.DADATA_API_KEY,
		TELEGRAM_MINI_APP_URL: process.env.TELEGRAM_MINI_APP_URL,
	})
	return {
		MENU_API_URL: MENU_API_URL.replace(/\/+$/, ''),
		CUSTOMER_PUBLIC_ORIGIN: CUSTOMER_PUBLIC_ORIGIN.replace(/\/+$/, ''),
		UPLOADS_ORIGIN: UPLOADS_ORIGIN?.replace(/\/+$/, ''),
		YANDEX_MAPS_API_KEY,
		DADATA_API_KEY,
		TELEGRAM_MINI_APP_URL,
	}
}
