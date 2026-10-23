import { serverEnv } from '@/shared/config'
import type { ContentPage } from '../model/types'

export type ApiBlock = {
	key: string
	page: string
	section: string
	label: string
	type: 'text' | 'lines' | 'richtext' | 'image'
	maxLength: number
	/** String for `text` and `richtext`, array for `lines`, `null` for `image`. */
	value: string | string[] | null
	/** `url` is a site path (`/images/…`) or a full URL of an uploaded image. */
	image: { url: string; alt: string; position: string } | null
	updatedAt: string
}

/** GET /content/blocks?page=…: the blocks of one page. */
export async function fetchContentBlocks(page: ContentPage): Promise<ApiBlock[]> {
	const response = await fetch(serverEnv().MENU_API_URL + `/content/blocks?page=${page}`, { next: { revalidate: 60 } })
	if (!response.ok) throw new Error(`GET /content/blocks?page=${page}: ${response.status}`)
	return (await response.json()) as ApiBlock[]
}
