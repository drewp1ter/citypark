import { useState, type ReactElement } from 'react'
import { formatRub } from '../../../shared/lib/format'
import { useMediaQuery } from '../../../shared/lib/use-media-query'
import type { ReportConfig } from '../model/reports'
import { buildTree, grandTotals, type Row, type TreeNode } from '../model/tree'

type Column = ReportConfig['columns'][number]

function formatValue(column: Column, value: number): string {
	return column.kind === 'money' ? formatRub(value) : value.toLocaleString('ru-RU')
}

/** Expanded nodes are remembered by their path of labels. */
function useExpanded() {
	const [open, setOpen] = useState<ReadonlySet<string>>(new Set())
	const toggle = (path: string) =>
		setOpen((prev) => {
			const next = new Set(prev)
			if (!next.delete(path)) next.add(path)
			return next
		})
	return { open, toggle }
}

type LevelProps = {
	nodes: readonly TreeNode[]
	columns: readonly Column[]
	open: ReadonlySet<string>
	toggle: (path: string) => void
	parent: string
	depth: number
}

function AccordionLevel({ nodes, columns, open, toggle, parent, depth }: LevelProps) {
	return (
		<ul className={depth === 0 ? 'flex flex-col gap-2' : 'mt-2 flex flex-col gap-2 border-l border-border pl-3'}>
			{nodes.map((node) => {
				const path = `${parent}/${node.label}`
				const expanded = open.has(path)
				const expandable = node.children.length > 0
				const head = (
					<>
						<span className='font-medium'>{node.label}</span>
						<span className='mt-1 flex flex-wrap gap-x-4 text-sm text-muted'>
							{columns.map((c) => (
								<span key={c.key}>
									{c.title}: {formatValue(c, node.totals[c.key] ?? 0)}
								</span>
							))}
						</span>
					</>
				)
				return (
					<li key={node.label} className='rounded-lg border border-border bg-surface p-3'>
						{expandable ? (
							<button
								type='button'
								aria-expanded={expanded}
								onClick={() => toggle(path)}
								className='flex w-full flex-col items-start text-left'
							>
								{head}
							</button>
						) : (
							<div className='flex flex-col items-start'>{head}</div>
						)}
						{expandable && expanded && (
							<AccordionLevel
								nodes={node.children}
								columns={columns}
								open={open}
								toggle={toggle}
								parent={path}
								depth={depth + 1}
							/>
						)}
					</li>
				)
			})}
		</ul>
	)
}

function tableRows({ nodes, columns, open, toggle, parent, depth }: LevelProps): ReactElement[] {
	return nodes.flatMap((node) => {
		const path = `${parent}/${node.label}`
		const expanded = open.has(path)
		const expandable = node.children.length > 0
		const row = (
			<tr key={path} className='border-t border-border'>
				<th scope='row' className='py-2 pr-4 text-left font-normal' style={{ paddingLeft: `${depth * 1.5}rem` }}>
					{expandable ? (
						<button
							type='button'
							aria-expanded={expanded}
							onClick={() => toggle(path)}
							className='flex items-center gap-2 font-medium'
						>
							<span aria-hidden='true'>{expanded ? '▾' : '▸'}</span>
							{node.label}
						</button>
					) : (
						<span className='pl-5'>{node.label}</span>
					)}
				</th>
				{columns.map((c) => (
					<td key={c.key} className='py-2 pl-4 text-right tabular-nums'>
						{formatValue(c, node.totals[c.key] ?? 0)}
					</td>
				))}
			</tr>
		)
		const children =
			expandable && expanded
				? tableRows({ nodes: node.children, columns, open, toggle, parent: path, depth: depth + 1 })
				: []
		return [row, ...children]
	})
}

export function ReportTree({ config, rows }: { config: ReportConfig; rows: readonly Row[] }) {
	const desktop = useMediaQuery('(min-width: 768px)')
	const { open, toggle } = useExpanded()
	const sumKeys = config.columns.map((c) => c.key)
	const tree = buildTree(rows, config.groupBy, sumKeys)
	const totals = grandTotals(tree, sumKeys)

	if (!desktop) {
		return (
			<div className='flex flex-col gap-3'>
				<AccordionLevel nodes={tree} columns={config.columns} open={open} toggle={toggle} parent='' depth={0} />
				<div className='flex flex-wrap gap-x-4 rounded-lg bg-surface-secondary p-3 font-medium'>
					<span>Итого</span>
					{config.columns.map((c) => (
						<span key={c.key}>
							{c.title}: {formatValue(c, totals[c.key] ?? 0)}
						</span>
					))}
				</div>
			</div>
		)
	}

	return (
		<div className='overflow-x-auto'>
			<table className='w-full text-sm'>
				<caption className='sr-only'>{config.title}</caption>
				<thead>
					<tr>
						<th scope='col' className='py-2 pr-4 text-left font-medium text-muted'>
							{config.groupBy.length > 0 ? 'Группа' : ''}
						</th>
						{config.columns.map((c) => (
							<th key={c.key} scope='col' className='py-2 pl-4 text-right font-medium text-muted'>
								{c.title}
							</th>
						))}
					</tr>
				</thead>
				<tbody>
					{tableRows({ nodes: tree, columns: config.columns, open, toggle, parent: '', depth: 0 })}
					<tr className='border-t-2 border-border font-semibold'>
						<th scope='row' className='py-2 pr-4 text-left'>
							Итого
						</th>
						{config.columns.map((c) => (
							<td key={c.key} className='py-2 pl-4 text-right tabular-nums'>
								{formatValue(c, totals[c.key] ?? 0)}
							</td>
						))}
					</tr>
				</tbody>
			</table>
		</div>
	)
}
