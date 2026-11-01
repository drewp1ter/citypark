import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { deliverySettingsQuery } from '../api'
import { DeliverySettingsFormView } from './delivery-settings-form'
export function DeliverySettingsCard() {
	const query = useQuery(deliverySettingsQuery)
	if (query.isPending)
		return <Skeleton className='mb-6 h-60 w-full rounded-xl' aria-label='Загрузка информации о доставке' />
	if (!query.data)
		return (
			<div role='alert' className='mb-6 rounded-xl border border-border p-4'>
				<p>{query.error?.message}</p>
				<Button onPress={() => void query.refetch()}>Повторить</Button>
			</div>
		)
	return <DeliverySettingsFormView settings={query.data} />
}
