import type { MetadataRoute } from 'next'
import { productionOrigin } from '@/shared/config'

// Let crawlers read page-level noindex directives, including payment returns.
export default function robots(): MetadataRoute.Robots {
	return { rules: { userAgent: '*', allow: '/' }, sitemap: `${productionOrigin}/sitemap.xml` }
}
