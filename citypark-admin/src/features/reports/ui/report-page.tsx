import { useQuery } from '@tanstack/react-query'
import { getRouteApi, Link, useNavigate } from '@tanstack/react-router'
import { moscowToday } from '../../../shared/lib/today'
import { PageHeader } from '../../../shared/ui/page-header'
import { hotelSalesQuery } from '../api'
import { resolvePeriod } from '../model/periods'
import { findReport } from '../model/reports'
import { PeriodPicker } from './period-picker'
import { ReportState } from './report-state'
import { ReportTree } from './report-tree'

const route = getRouteApi('/_app/reports_/$reportId')

function NotFound() {
	return (
		<>
			<PageHeader title='Отчёты' />
			<p className='mb-3 text-muted'>Отчёт не найден</p>
			<Link to='/reports' className='font-medium text-accent hover:underline'>
				Ко всем отчётам
			</Link>
		</>
	)
}

export function ReportPage({ reportId }: { reportId: string }) {
	const config = findReport(reportId)
	return config === undefined ? <NotFound /> : <KnownReport config={config} />
}

function KnownReport({ config }: { config: NonNullable<ReturnType<typeof findReport>> }) {
	const search = route.useSearch()
	const navigate = useNavigate({ from: '/reports/$reportId' })
	const period = resolvePeriod(search, moscowToday())
	// One endpoint today; a new endpoint in the config means a new branch here.
	const query = useQuery(hotelSalesQuery(period))

	return (
		<>
			<PageHeader
				title={config.title}
				actions={
					<Link to='/reports' search={period} className='text-sm text-accent hover:underline'>
						Все отчёты
					</Link>
				}
			/>
			<PeriodPicker period={period} onChange={(next) => void navigate({ search: next })} />
			<ReportState query={query} isEmpty={(data) => data.rows.length === 0}>
				{(data) => <ReportTree config={config} rows={data.rows} />}
			</ReportState>
		</>
	)
}
