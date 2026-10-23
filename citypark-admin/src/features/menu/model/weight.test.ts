import { describe, expect, it } from 'vitest'
import { formatWeight } from './weight'

const NBSP = String.fromCharCode(0xa0)
describe('formatWeight', () => {
	it('shows kilograms below 1 as grams', () => {
		expect(formatWeight(0.35, 'kg')).toBe('350' + NBSP + 'г')
		expect(formatWeight(0.35, '')).toBe('350' + NBSP + 'г')
	})

	it('shows 1 kg and more in kilograms', () => {
		expect(formatWeight(1.5, 'кг')).toBe('1,5' + NBSP + 'кг')
		expect(formatWeight(1, 'kg')).toBe('1' + NBSP + 'кг')
	})

	it('shows another unit as given', () => {
		expect(formatWeight(2, 'шт')).toBe('2 шт')
	})

	it('returns an empty string without a weight', () => {
		expect(formatWeight(0, 'kg')).toBe('')
	})
})
