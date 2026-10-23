import { describe, expect, it } from 'vitest'
import { insertPlaceholder } from './placeholders'
import { isDirty, mapFieldErrors, toDraft, toRequest } from './draft'
import type { ApiBlock } from './sections'

const base: ApiBlock = {
	key: 'k',
	page: 'home',
	section: 'S',
	label: 'L',
	type: 'text',
	maxLength: 80,
	value: 'Привет',
	image: null,
	updatedAt: '2026-10-22T09:05:00Z',
}
const lines: ApiBlock = { ...base, type: 'lines', maxLength: 0, value: ['a', 'b'] }
const image: ApiBlock = {
	...base,
	type: 'image',
	maxLength: 0,
	value: null,
	image: { url: '/images/a.jpg', alt: 'Фото', position: '' },
}

describe('draft', () => {
	it('starts from the saved value', () => {
		expect(toDraft(base)).toBe('Привет')
		expect(toDraft(lines)).toEqual(['a', 'b'])
		expect(toDraft(image)).toEqual({ url: '/images/a.jpg', alt: 'Фото', position: '' })
	})

	it('detects changes', () => {
		expect(isDirty(base, 'Привет')).toBe(false)
		expect(isDirty(base, 'Привет!')).toBe(true)
		expect(isDirty(lines, ['a', 'b'])).toBe(false)
		expect(isDirty(lines, ['b', 'a'])).toBe(true)
		expect(isDirty(image, { url: '/images/a.jpg', alt: 'Фото', position: '10% 20%' })).toBe(true)
	})

	it('builds the request body by type', () => {
		expect(toRequest(base, 'x')).toEqual({ value: 'x' })
		expect(toRequest(lines, ['x'])).toEqual({ value: ['x'] })
		expect(toRequest(image, { url: '/u', alt: 'a', position: '' })).toEqual({
			image: { url: '/u', alt: 'a', position: '' },
		})
	})

	it('strips the "image." prefix from field errors', () => {
		expect(mapFieldErrors({ 'image.url': 'нужен', value: 'плохо' })).toEqual({ url: 'нужен', value: 'плохо' })
	})
})

describe('insertPlaceholder', () => {
	it('inserts at the caret', () => {
		expect(insertPlaceholder('от  в сутки', 'minRoomPrice', 3, 3)).toEqual({
			value: 'от {{minRoomPrice}} в сутки',
			caret: 3 + '{{minRoomPrice}}'.length,
		})
	})

	it('replaces the selection, also a backwards one', () => {
		expect(insertPlaceholder('abcdef', 'x', 4, 2).value).toBe('ab{{x}}ef')
	})
})
