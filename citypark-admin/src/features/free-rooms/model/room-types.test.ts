import { describe, expect, it } from 'vitest'
import type { components } from '../../../shared/api/schema'
import { isStale, ROOM_TYPES, TOTAL_ROOMS } from './room-types'

describe('ROOM_TYPES', () => {
	it('totals add up to 12', () => {
		expect(TOTAL_ROOMS).toBe(12)
	})

	it('has a label for every API key, in order', () => {
		const sample: components['schemas']['HotelRooms'] = {
			vip: 0,
			singleBalcony: 0,
			single: 0,
			doubleTwin: 0,
			doubleEuro: 0,
		}
		expect(ROOM_TYPES.map((t) => t.key).sort()).toEqual(Object.keys(sample).sort())
		expect(ROOM_TYPES.every((t) => t.label !== '')).toBe(true)
		expect(ROOM_TYPES.map((t) => t.key)).toEqual(['vip', 'singleBalcony', 'single', 'doubleTwin', 'doubleEuro'])
	})
})

describe('isStale', () => {
	const now = Date.parse('2026-10-23T12:00:00Z')
	it('is false within 24 hours', () => {
		expect(isStale('2026-10-22T12:00:01Z', now)).toBe(false)
	})
	it('is true after 24 hours', () => {
		expect(isStale('2026-10-22T11:59:59Z', now)).toBe(true)
	})
})
