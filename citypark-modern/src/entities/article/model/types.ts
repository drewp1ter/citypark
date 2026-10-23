import type { Photo } from '@/shared/lib'

export type NewsItem = {
	/** Used in /news/[slug]. */
	slug: string
	/** ISO date, "2026-09-07". Show it with `formatDate`. */
	date: string
	title: string
	/** Shorter title used on the home page and the mobile news page. */
	shortTitle?: string
	/** Teaser under the title of the latest news. */
	excerpt?: string
	photo: Photo
	/** Sanitized HTML of the article text; `''` when there is none. Empty in list items. */
	bodyHtml: string
}
