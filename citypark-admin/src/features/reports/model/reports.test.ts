import { describe, expect, it } from 'vitest'
import { findReport, reports } from './reports'

describe('reports config', () => {
	it('has unique ids', () => {
		const ids = reports.map((r) => r.id)
		expect(new Set(ids).size).toBe(ids.length)
	})

	it('has at least one column and one tree level per report', () => {
		for (const report of reports) {
			expect(report.columns.length).toBeGreaterThan(0)
			expect(report.groupBy.length).toBeGreaterThan(0)
		}
	})

	it('finds a report by id', () => {
		expect(findReport('hotel-sales')?.title).toBe('Сдача номеров')
		expect(findReport('nope')).toBeUndefined()
	})
})
