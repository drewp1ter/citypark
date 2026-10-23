import { cache } from 'react'
import {
	defaultSiteSettings,
	serverEnv,
	toSiteSettings,
	type ApiSiteSettings,
	type SiteSettings,
} from '@/shared/config'

/** GET /settings (revalidate 60); on any error logs it and returns `defaultSiteSettings`, so contacts never disappear. */
export const getSiteSettings = cache(async (): Promise<SiteSettings> => {
	try {
		const response = await fetch(serverEnv().MENU_API_URL + '/settings', { next: { revalidate: 60 } })
		if (!response.ok) throw new Error(`GET /settings: ${response.status}`)
		return toSiteSettings((await response.json()) as ApiSiteSettings)
	} catch (error) {
		console.error('Site settings: using the built-in contacts.', error)
		return defaultSiteSettings
	}
})
