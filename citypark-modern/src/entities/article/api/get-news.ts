import { news } from '../model/news.data'
import type { NewsItem } from '../model/types'

// Reads go through these functions only; swap the bodies when a CMS appears.

export async function getNews(): Promise<readonly NewsItem[]> {
	return news
}

export async function getNewsItem(slug: string): Promise<NewsItem | undefined> {
	return news.find((item) => item.slug === slug)
}
