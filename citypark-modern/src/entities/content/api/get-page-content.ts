import { fillPlaceholders, typo } from '@/shared/lib'
import { blocks } from '../model/blocks.data'
import type { Block, ContentPage, PageContent } from '../model/types'
import { fetchContentBlocks, type ApiBlock } from './content-api'

const reported = new Set<string>()

function missing(key: string): void {
	if (reported.has(key)) return
	reported.add(key)
	console.error(`content block missing: ${key}`)
}

function fromApi(block: ApiBlock): Block | null {
	const base = { key: block.key, section: block.section, label: block.label }
	switch (block.type) {
		case 'text':
		case 'richtext':
			return typeof block.value === 'string' ? { ...base, type: block.type, value: block.value } : null
		case 'lines':
			return Array.isArray(block.value) ? { ...base, type: 'lines', value: block.value } : null
		case 'image':
			return block.image
				? {
						...base,
						type: 'image',
						image: { src: block.image.url, alt: block.image.alt, position: block.image.position || undefined },
					}
				: null
	}
}

const reportedMissing = new Set<string>()

/** Copy of one page from the backend. A block the API lacks (or a whole failed request) falls back to the static copy. */
export async function getPageContent(page: ContentPage): Promise<PageContent> {
	const byKey = new Map<string, Block>()
	for (const block of blocks) {
		if (block.key.startsWith(`${page}.`)) byKey.set(block.key, block)
	}
	try {
		const fromBackend = new Set<string>()
		for (const apiBlock of await fetchContentBlocks(page)) {
			const block = fromApi(apiBlock)
			if (!block) continue
			byKey.set(block.key, block)
			fromBackend.add(block.key)
		}
		for (const key of byKey.keys()) {
			if (fromBackend.has(key) || reportedMissing.has(key)) continue
			reportedMissing.add(key)
			console.error(`content block missing in API: ${key}`)
		}
	} catch (error) {
		console.error(`content blocks (${page}): using static data:`, error)
	}

	function find<T extends Block['type']>(key: string, type: T): Extract<Block, { type: T }> | null {
		const block = byKey.get(key)
		if (!block || block.type !== type) {
			missing(key)
			return null
		}
		return block as Extract<Block, { type: T }>
	}

	return {
		text: (key, values = {}) => {
			const block = find(key, 'text')
			return block ? fillPlaceholders(block.value, values) : ''
		},
		lines: (key) => find(key, 'lines')?.value.map((line) => typo(line)) ?? [],
		html: (key, values = {}) => {
			const block = find(key, 'richtext')
			return block ? fillPlaceholders(block.value, values, { typography: false }) : ''
		},
		image: (key) => find(key, 'image')?.image ?? null,
	}
}
