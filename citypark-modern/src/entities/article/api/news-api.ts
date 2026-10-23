import { serverEnv } from '@/shared/config'

export type ApiNewsItem = {
	id: number
	slug: string
	/** ISO date, "2026-09-07". */
	date: string
	title: string
	shortTitle: string
	excerpt: string
	/** HTML of the article profile. Only in `GET /news/{slug}`. */
	body?: string
	/** `url` is `""` (no photo), a site path (`/images/…`) or a full URL of an uploaded image. */
	photo: { url: string; alt: string; position: string }
	updatedAt: string
}

const PAGE_SIZE = 100

/** GET /news: every page, newest first. Items have no `body`. */
export async function fetchNews(): Promise<ApiNewsItem[]> {
	const items: ApiNewsItem[] = []
	for (let page = 1; ; page++) {
		const response = await fetch(`${serverEnv().MENU_API_URL}/news?page=${page}&pageSize=${PAGE_SIZE}`, {
			next: { revalidate: 60 },
		})
		if (!response.ok) throw new Error(`GET /news: ${response.status}`)
		const data = (await response.json()) as { news: ApiNewsItem[]; count: number }
		items.push(...data.news)
		if (data.news.length === 0 || items.length >= data.count) return items
	}
}

/** GET /news/{slug}; `undefined` on 404. */
export async function fetchNewsItem(slug: string): Promise<ApiNewsItem | undefined> {
	const response = await fetch(`${serverEnv().MENU_API_URL}/news/${encodeURIComponent(slug)}`, {
		next: { revalidate: 60 },
	})
	if (response.status === 404) return undefined
	if (!response.ok) throw new Error(`GET /news/${slug}: ${response.status}`)
	return (await response.json()) as ApiNewsItem
}
