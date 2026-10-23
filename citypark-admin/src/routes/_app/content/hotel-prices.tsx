import { createFileRoute } from '@tanstack/react-router'
import { HotelPricesPage } from '../../../features/hotel-prices/ui/hotel-prices-page'

export const Route = createFileRoute('/_app/content/hotel-prices')({
	component: HotelPricesPage,
})
