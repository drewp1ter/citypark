import type { MetadataRoute } from 'next'

// Let crawlers read page-level noindex directives, including payment returns.
export default function robots(): MetadataRoute.Robots {
	return { rules: { userAgent: '*', allow: '/' } }
}
