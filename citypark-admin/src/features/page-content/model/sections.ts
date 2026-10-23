import type { components } from '../../../shared/api/schema'

export type ApiBlock = components['schemas']['ContentBlock']
export type BlockSection = { name: string; blocks: ApiBlock[] }

const MOBILE_SUFFIX = '-mobile'

/** Groups blocks by section in the order they come from the API; pairs "<key>-mobile" right after "<key>". */
export function toSections(blocks: ApiBlock[]): BlockSection[] {
	const sections: BlockSection[] = []
	for (const block of blocks) {
		let section = sections.find((s) => s.name === block.section)
		if (section === undefined) {
			section = { name: block.section, blocks: [] }
			sections.push(section)
		}
		section.blocks.push(block)
	}
	for (const section of sections) section.blocks = pairMobile(section.blocks)
	return sections
}

function pairMobile(blocks: ApiBlock[]): ApiBlock[] {
	const keys = new Set(blocks.map((b) => b.key))
	const mobileOf = new Map<string, ApiBlock>()
	for (const block of blocks) {
		if (!block.key.endsWith(MOBILE_SUFFIX)) continue
		const main = block.key.slice(0, -MOBILE_SUFFIX.length)
		if (keys.has(main)) mobileOf.set(main, block)
	}
	const paired = new Set(mobileOf.values())
	const result: ApiBlock[] = []
	for (const block of blocks) {
		if (paired.has(block)) continue
		result.push(block)
		const mobile = mobileOf.get(block.key)
		if (mobile !== undefined) result.push(mobile)
	}
	return result
}
