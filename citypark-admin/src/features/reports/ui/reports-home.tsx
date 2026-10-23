import { useQuery } from '@tanstack/react-query'
import { getRouteApi, Link, useNavigate } from '@tanstack/react-router'
import { formatRub } from '../../../shared/lib/format'
import { moscowToday } from '../../../shared/lib/today'
import { PageHeader } from '../../../shared/ui/page-header'
import { hotelDailyQuery } from '../api'
import { resolvePeriod, type Period } from '../model/periods'
import { reports } from '../model/reports'
import { dashboardTiles } from '../model/tiles'
import { DailyChart } from './daily-chart'
import { PeriodPicker } from './period-picker'
import { ReportState } from './report-state'

const route = getRouteApi('/_app/reports')

function Tile({ title, value }: { title: string; value: number | null }) {
	return (
		<li className='rounded-xl border border-border bg-surface p-4'>
			<div className='text-sm text-muted'>{title}</div>
			<div className='mt-1 text-xl font-semibold'>{value === null ? '—' : formatRub(value)}</div>
		</li>
	)
}

export function ReportsHome() {
	const search = route.useSearch()
	const navigate = useNavigate({ from: '/reports' })
	const today = moscowToday()
	const period = resolvePeriod(search, today)
	const query = useQuery(hotelDailyQuery(period))

	return (
		<>
			<PageHeader title='Отчёты' />
			<PeriodPicker period={period} onChange={(next: Period) => void navigate({ search: next })} />
			<div className='mb-8'>
				<ReportState query={query} isEmpty={(data) => data.days.length === 0}>
					{(data) => {
						const tiles = dashboardTiles(data.days, period, today)
						return (
							<div className='flex flex-col gap-4'>
								<ul className='grid grid-cols-2 gap-3 lg:grid-cols-4'>
									<Tile title='Сегодня' value={tiles.today} />
									<Tile title='Вчера' value={tiles.yesterday} />
									<Tile title='7 дней' value={tiles.week} />
									<Tile title='За период' value={data.total} />
								</ul>
								<DailyChart days={data.days} />
							</div>
						)
					}}
				</ReportState>
			</div>
			<h2 className='mb-3 text-lg font-semibold'>Отчёты</h2>
			<ul className='flex flex-col gap-2'>
				{reports.map((r) => (
					<li key={r.id}>
						<Link
							to='/reports/$reportId'
							params={{ reportId: r.id }}
							search={period}
							className='font-medium text-accent hover:underline'
						>
							{r.title}
						</Link>
					</li>
				))}
			</ul>
		</>
	)
}
