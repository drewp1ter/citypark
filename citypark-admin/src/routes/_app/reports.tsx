import { createFileRoute } from '@tanstack/react-router'
import { ReportsHome } from '../../features/reports/ui/reports-home'
import { parseReportSearch } from '../../features/reports/model/periods'

export const Route = createFileRoute('/_app/reports')({
	validateSearch: parseReportSearch,
	component: ReportsHome,
})
