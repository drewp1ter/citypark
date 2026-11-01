import { createFileRoute } from '@tanstack/react-router'
import { DeliveryPricesPage } from '../../features/delivery-prices/ui/delivery-prices-page'

import { WorkingHoursCard } from '../../features/working-hours/ui/working-hours-card'
import { DeliverySettingsCard } from '../../features/delivery-settings/ui/delivery-settings-card'

export const Route = createFileRoute('/_app/delivery')({
	component: () => (
		<DeliveryPricesPage>
			<WorkingHoursCard />
			<DeliverySettingsCard />
		</DeliveryPricesPage>
	),
})
