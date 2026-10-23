export type Row = Record<string, string | number>

export type TreeNode = {
	label: string
	totals: Record<string, number>
	children: TreeNode[]
}

function addTo(totals: Record<string, number>, row: Row, sumKeys: readonly string[]) {
	for (const key of sumKeys) {
		const value = row[key]
		totals[key] = (totals[key] ?? 0) + (typeof value === 'number' ? value : 0)
	}
}

function zeroTotals(sumKeys: readonly string[]): Record<string, number> {
	return Object.fromEntries(sumKeys.map((key) => [key, 0]))
}

/**
 * Groups rows into nested nodes by `groupBy` (outermost field first). Every node holds the sums of `sumKeys` over its
 * rows; the nodes of the last level are the leaves. Order follows the first appearance in `rows`.
 */
export function buildTree(rows: readonly Row[], groupBy: readonly string[], sumKeys: readonly string[]): TreeNode[] {
	const [field, ...rest] = groupBy
	if (field === undefined) return []
	const groups = new Map<string, Row[]>()
	for (const row of rows) {
		const label = String(row[field] ?? '')
		const group = groups.get(label)
		if (group === undefined) groups.set(label, [row])
		else group.push(row)
	}
	return [...groups].map(([label, groupRows]) => {
		const totals = zeroTotals(sumKeys)
		for (const row of groupRows) addTo(totals, row, sumKeys)
		return { label, totals, children: buildTree(groupRows, rest, sumKeys) }
	})
}

/** The sum of the top-level nodes: the grand total row. */
export function grandTotals(nodes: readonly TreeNode[], sumKeys: readonly string[]): Record<string, number> {
	const totals = zeroTotals(sumKeys)
	for (const node of nodes) {
		for (const key of sumKeys) totals[key] = (totals[key] ?? 0) + (node.totals[key] ?? 0)
	}
	return totals
}
