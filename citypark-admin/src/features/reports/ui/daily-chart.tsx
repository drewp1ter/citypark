import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { formatDate, formatRub } from '../../../shared/lib/format'
import { shortDate, type Day } from '../model/tiles'

const axisTick = { fill: 'var(--muted)', fontSize: 12 }

function ChartTooltip({
	active,
	payload,
	label,
}: {
	active?: boolean
	payload?: { value?: unknown }[]
	label?: unknown
}) {
	const value = payload?.[0]?.value
	if (active !== true || typeof value !== 'number' || typeof label !== 'string') return null
	return (
		<div className='rounded-lg border border-border bg-surface px-3 py-2 text-sm shadow-md'>
			<div className='text-muted'>{formatDate(label)}</div>
			<div className='font-medium'>{formatRub(value)}</div>
		</div>
	)
}

/** Revenue per day. Colours come from the HeroUI theme variables, so both themes work. */
export function DailyChart({ days }: { days: readonly Day[] }) {
	return (
		<div className='h-64 w-full' role='img' aria-label='График выручки по дням'>
			<ResponsiveContainer width='100%' height='100%'>
				<AreaChart data={[...days]} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
					<defs>
						<linearGradient id='daily-fill' x1='0' y1='0' x2='0' y2='1'>
							<stop offset='0%' stopColor='var(--accent)' stopOpacity={0.5} />
							<stop offset='100%' stopColor='var(--accent)' stopOpacity={0.05} />
						</linearGradient>
					</defs>
					<CartesianGrid stroke='var(--border)' strokeDasharray='3 3' vertical={false} />
					<XAxis dataKey='date' tickFormatter={shortDate} tick={axisTick} stroke='var(--border)' minTickGap={24} />
					<YAxis
						width={56}
						tick={axisTick}
						stroke='var(--border)'
						tickFormatter={(v: number) => v.toLocaleString('ru-RU')}
					/>
					<Tooltip content={<ChartTooltip />} />
					<Area
						type='monotone'
						dataKey='sum'
						stroke='var(--accent)'
						strokeWidth={2}
						fill='url(#daily-fill)'
						isAnimationActive={false}
					/>
				</AreaChart>
			</ResponsiveContainer>
		</div>
	)
}
