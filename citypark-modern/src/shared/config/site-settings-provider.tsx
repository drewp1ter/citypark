'use client'

import { createContext, useContext, type ReactNode } from 'react'
import { defaultSiteSettings, type SiteSettings } from './site-settings'

const SiteSettingsContext = createContext<SiteSettings>(defaultSiteSettings)

/** Gives client components the contacts the root layout fetched on the server. */
export function SiteSettingsProvider({ value, children }: { value: SiteSettings; children: ReactNode }) {
	return <SiteSettingsContext value={value}>{children}</SiteSettingsContext>
}

export function useSiteSettings(): SiteSettings {
	return useContext(SiteSettingsContext)
}
