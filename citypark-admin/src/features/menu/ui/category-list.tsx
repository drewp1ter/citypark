import { HIDDEN_PARAM, type MenuGroup } from '../model/group'

type Props = {
	groups: MenuGroup[]
	total: number
	/** The `category` search param. */
	selected: number | undefined
	onSelect: (category: number | undefined) => void
}

const base =
	'flex items-center justify-between gap-2 rounded-lg px-3 py-2 text-left text-sm transition-colors focus-visible:outline-2 focus-visible:outline-focus'

/** A sidebar on desktop and a row of scrolling chips on a phone. */
export function CategoryList({ groups, total, selected, onSelect }: Props) {
	const items = [
		{ param: undefined, name: 'Все', count: total, stoppedCount: 0 },
		...groups.map((g) => ({
			param: g.id === 'hidden' ? HIDDEN_PARAM : g.id,
			name: g.name,
			count: g.dishes.length,
			stoppedCount: g.dishes.filter((dish) => dish.isOnStopList).length,
		})),
	]
	return (
		<nav aria-label='Категории меню'>
			<ul className='flex gap-2 overflow-x-auto pb-2 lg:flex-col lg:gap-1 lg:overflow-visible lg:pb-0'>
				{items.map((item) => {
					const active = item.param === selected
					return (
						<li key={item.param ?? 'all'} className='shrink-0'>
							<button
								type='button'
								aria-label={
									item.stoppedCount > 0 ? `${item.name} ${item.count}. В стоп-листе: ${item.stoppedCount}` : undefined
								}
								aria-pressed={active}
								onClick={() => onSelect(item.param)}
								className={`${base} w-full whitespace-nowrap lg:whitespace-normal ${
									active ? 'bg-accent text-accent-foreground' : 'bg-default text-foreground hover:bg-default-hover'
								}`}
							>
								<span>
									{item.name} <span className={active ? '' : 'text-muted'}>{item.count}</span>
								</span>
								{item.stoppedCount > 0 && (
									<span
										aria-hidden='true'
										className={`grid aspect-square min-w-5 shrink-0 place-items-center rounded-full px-1 text-xs leading-none font-medium ${
											active ? 'bg-warning text-warning-foreground' : 'bg-warning-soft text-warning-soft-foreground'
										}`}
									>
										{item.stoppedCount}
									</span>
								)}
							</button>
						</li>
					)
				})}
			</ul>
		</nav>
	)
}
