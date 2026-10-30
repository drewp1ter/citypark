import { describe, expect, it } from 'vitest'
import { categoryDays } from './category-hours'

describe('categoryDays', () => {
	it('defaults to every day', () => expect(categoryDays('')).toEqual(Array(7).fill(true)))
	it('restores selected weekdays', () =>
		expect(categoryDays('Пн, Ср, Вс')).toEqual([true, false, true, false, false, false, true]))
})
