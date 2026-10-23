import { fillPlaceholders, typo } from '@/shared/lib'
import { blocks } from '../model/blocks.data'
import type { Block, ContentPage, PageContent } from '../model/types'

const reported = new Set<string>()

function missing(key: string): void {
	if (reported.has(key)) return
	reported.add(key)
	console.error(`content block missing: ${key}`)
}

/** Copy of one page. Later this reads the blocks from citypark-go (task 20); the accessors stay the same. */
export async function getPageContent(page: ContentPage): Promise<PageContent> {
	const byKey = new Map<string, Block>()
	for (const block of blocks) {
		if (block.key.startsWith(`${page}.`)) byKey.set(block.key, block)
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
