import { Chip } from '@heroui/react'
import { statusColor, statusLabel } from '../lib/order-labels'

export function StatusChip({ status }: { status: string }) {
	return (
		<Chip color={statusColor(status)} variant='soft' size='sm'>
			{statusLabel(status)}
		</Chip>
	)
}
