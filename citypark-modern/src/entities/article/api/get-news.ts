import { PLACEHOLDER_PHOTO } from '@/shared/lib'
import { fetchNews, fetchNewsItem, type ApiNewsItem } from './news-api'
import type { NewsItem } from '../model/types'

// Reads go through these functions only. News come from the backend.

function toNewsItem(item: ApiNewsItem): NewsItem {
	return {
		slug: item.slug,
		date: item.date,
		title: item.title,
		shortTitle: item.shortTitle || undefined,
		excerpt: item.excerpt || undefined,
		photo: item.photo.url
			? { src: item.photo.url, alt: item.photo.alt, position: item.photo.position || undefined }
			: { src: PLACEHOLDER_PHOTO, alt: item.title },
		bodyHtml: item.body ?? '',
	}
}

/** Newest first. Reads every page of GET /news (pageSize 100). Errors are left to the caller. */
export async function getNews(): Promise<readonly NewsItem[]> {
	return (await fetchNews()).map(toNewsItem)
}

/** `undefined` when there is no such item. Other errors are left to the caller. */
export async function getNewsItem(slug: string): Promise<NewsItem | undefined> {
	const item = await fetchNewsItem(slug)
	return item && toNewsItem(item)
}
