import { createFileRoute } from '@tanstack/react-router'
import { DeliveryPricesPage } from '../../features/delivery-prices/ui/delivery-prices-page'

export const Route = createFileRoute('/_app/delivery')({
	component: DeliveryPricesPage,
})
