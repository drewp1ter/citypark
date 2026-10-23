import type { PhotoValue } from '../../../shared/ui/image-field'
import type { components } from '../../../shared/api/schema'
import type { ApiBlock } from './sections'

export type BlockDraft = string | string[] | PhotoValue
type Input = components['schemas']['ContentBlockInput']

export const MAX_LINES = 50

/** The editable value of a block, as the editor of its type works with it. */
export function toDraft(block: ApiBlock): BlockDraft {
	if (block.type === 'image') {
		return { url: block.image?.url ?? '', alt: block.image?.alt ?? '', position: block.image?.position ?? '' }
	}
	if (block.type === 'lines') return Array.isArray(block.value) ? [...block.value] : []
	return typeof block.value === 'string' ? block.value : ''
}

export function isDirty(block: ApiBlock, draft: BlockDraft): boolean {
	return JSON.stringify(draft) !== JSON.stringify(toDraft(block))
}

export function toRequest(block: ApiBlock, draft: BlockDraft): Input {
	if (block.type === 'image') {
		const photo = draft as PhotoValue
		return { image: { url: photo.url, alt: photo.alt, position: photo.position } }
	}
	return { value: draft as string | string[] }
}

/** The backend names image fields "image.url"; the image editor wants "url". Other field errors keep their names. */
export function mapFieldErrors(fields: Record<string, string>): Record<string, string> {
	const mapped: Record<string, string> = {}
	for (const [name, text] of Object.entries(fields)) {
		mapped[name.startsWith('image.') ? name.slice('image.'.length) : name] = text
	}
	return mapped
}
