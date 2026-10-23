import { createFileRoute } from '@tanstack/react-router'
import { parseReportSearch } from '../../features/reports/model/periods'
import { ReportPage } from '../../features/reports/ui/report-page'

export const Route = createFileRoute('/_app/reports_/$reportId')({
	validateSearch: parseReportSearch,
	component: ReportRoute,
})

function ReportRoute() {
	const { reportId } = Route.useParams()
	return <ReportPage reportId={reportId} />
}
