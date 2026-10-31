import { createFileRoute } from '@tanstack/react-router'
import { parseCampaignSearch } from '../../features/promo-codes/model/form'
import { PromoCodesPage } from '../../features/promo-codes/ui/promo-codes-page'

export const Route = createFileRoute('/_app/promo-codes')({
	validateSearch: parseCampaignSearch,
	component: PromoCodesPage,
})
