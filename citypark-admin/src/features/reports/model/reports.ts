export type ReportConfig = {
	/** URL segment */
	id: string
	title: string
	endpoint: '/reports/hotel/sales'
	/** Row fields forming the tree levels, outermost first. */
	groupBy: readonly string[]
	columns: readonly { key: string; title: string; kind: 'count' | 'money' }[]
}

/** A new report is one more entry here. */
export const reports: readonly ReportConfig[] = [
	{
		id: 'hotel-sales',
		title: 'Сдача номеров',
		endpoint: '/reports/hotel/sales',
		groupBy: ['organization', 'room'],
		columns: [
			{ key: 'count', title: 'Кол-во', kind: 'count' },
			{ key: 'sum', title: 'Сумма', kind: 'money' },
		],
	},
]

export function findReport(id: string): ReportConfig | undefined {
	return reports.find((r) => r.id === id)
}
