const DEFAULT_SITE_URL = 'http://localhost:3001'

/** Address of the public site, without a trailing slash. */
export function siteUrl(): string {
	return (import.meta.env.VITE_SITE_URL ?? DEFAULT_SITE_URL).replace(/\/+$/, '')
}

/** Address to show a photo from: site photos (`/images/...`) live on the public site, everything else is used as is. */
export function imageSrc(url: string): string {
	if (url.startsWith('/images/')) return siteUrl() + url
	return url
}
