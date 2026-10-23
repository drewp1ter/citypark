import { describe, expect, it } from 'vitest'
import { buildTree, grandTotals } from './tree'

const rows = [
	{ organization: 'ИП Иванов', room: 'Стандарт', count: 2, sum: 4000 },
	{ organization: 'ООО «Олимп»', room: 'VIP-номер', count: 3, sum: 12000.5 },
	{ organization: 'ООО «Олимп»', room: 'Люкс', count: 1, sum: 9000 },
]
const groupBy = ['organization', 'room']
const sumKeys = ['count', 'sum']

describe('buildTree', () => {
	it('sums every level', () => {
		const tree = buildTree(rows, groupBy, sumKeys)
		expect(tree.map((n) => n.label)).toEqual(['ИП Иванов', 'ООО «Олимп»'])
		expect(tree[0]?.totals).toEqual({ count: 2, sum: 4000 })
		expect(tree[1]?.totals).toEqual({ count: 4, sum: 21000.5 })
		expect(tree[1]?.children.map((n) => n.label)).toEqual(['VIP-номер', 'Люкс'])
		expect(tree[1]?.children[1]?.totals).toEqual({ count: 1, sum: 9000 })
	})

	it('makes leaves of the last level', () => {
		const leaf = buildTree(rows, groupBy, sumKeys)[0]?.children[0]
		expect(leaf?.children).toEqual([])
	})

	it('merges rows with the same label', () => {
		const tree = buildTree(
			[...rows, { organization: 'ИП Иванов', room: 'Стандарт', count: 1, sum: 100 }],
			groupBy,
			sumKeys
		)
		expect(tree[0]?.children).toHaveLength(1)
		expect(tree[0]?.children[0]?.totals).toEqual({ count: 3, sum: 4100 })
	})

	it('gives [] for no rows', () => {
		expect(buildTree([], groupBy, sumKeys)).toEqual([])
	})
})

describe('grandTotals', () => {
	it('sums the top level', () => {
		expect(grandTotals(buildTree(rows, groupBy, sumKeys), sumKeys)).toEqual({ count: 6, sum: 25000.5 })
	})

	it('is zero for an empty tree', () => {
		expect(grandTotals([], sumKeys)).toEqual({ count: 0, sum: 0 })
	})
})
