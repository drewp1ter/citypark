import { describe, expect, it } from 'vitest'
import { parsePosition, toPosition } from './focal-point'

describe('toPosition', () => {
	it('rounds ratios to whole percents', () => {
		expect(toPosition(0.374, 0.6)).toBe('37% 60%')
	})
	it('clamps to 0..1', () => {
		expect(toPosition(-1, 2)).toBe('0% 100%')
	})
})

describe('parsePosition', () => {
	it('reads x and y', () => {
		expect(parsePosition('20% 80%')).toEqual({ x: 20, y: 80 })
	})
	it('gives the centre for an empty or invalid string', () => {
		expect(parsePosition('')).toEqual({ x: 50, y: 50 })
		expect(parsePosition('left top')).toEqual({ x: 50, y: 50 })
		expect(parsePosition('120% 10%')).toEqual({ x: 50, y: 50 })
	})
})
