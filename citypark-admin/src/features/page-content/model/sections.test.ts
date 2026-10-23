import { describe, expect, it } from 'vitest'
import { toSections, type ApiBlock } from './sections'

function block(key: string, section: string): ApiBlock {
	return {
		key,
		page: 'home',
		section,
		label: key,
		type: 'text',
		maxLength: 80,
		value: '',
		image: null,
		updatedAt: '2026-10-22T09:05:00Z',
	}
}

const keys = (blocks: ApiBlock[] | undefined) => (blocks ?? []).map((b) => b.key)

describe('toSections', () => {
	it('groups by section and keeps the API order', () => {
		const sections = toSections([block('a', 'Один'), block('b', 'Два'), block('c', 'Один'), block('d', 'Два')])
		expect(sections.map((s) => s.name)).toEqual(['Один', 'Два'])
		expect(keys(sections[0]?.blocks)).toEqual(['a', 'c'])
		expect(keys(sections[1]?.blocks)).toEqual(['b', 'd'])
	})

	it('puts "<key>-mobile" right after "<key>"', () => {
		const [section] = toSections([block('x.lead', 'S'), block('x.title', 'S'), block('x.lead-mobile', 'S')])
		expect(keys(section?.blocks)).toEqual(['x.lead', 'x.lead-mobile', 'x.title'])
	})

	it('leaves a "-mobile" block without its main block where it is', () => {
		const [section] = toSections([block('x.a', 'S'), block('x.check-in-mobile', 'S'), block('x.b', 'S')])
		expect(keys(section?.blocks)).toEqual(['x.a', 'x.check-in-mobile', 'x.b'])
	})

	it('does not pair across sections', () => {
		const sections = toSections([block('x.lead', 'A'), block('x.lead-mobile', 'B')])
		expect(keys(sections[0]?.blocks)).toEqual(['x.lead'])
		expect(keys(sections[1]?.blocks)).toEqual(['x.lead-mobile'])
	})

	it('returns no sections for no blocks', () => {
		expect(toSections([])).toEqual([])
	})
})
