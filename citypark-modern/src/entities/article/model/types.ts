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
	/** Article text, one string per paragraph. Not written yet. */
	body?: readonly string[]
}

export type BlogPost = {
	slug: string
	title: string
	/** "Итальянский омлет" */
	subtitle: string
	/** Shorter title for the mobile news page. */
	shortTitle?: string
	photo: Photo
	body?: readonly string[]
}

export type BlogAuthor = { name: string; photo: Photo }
