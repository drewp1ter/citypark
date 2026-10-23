import { Chip } from '@heroui/react'
import { paymentLabel, statusColor, statusLabel } from '../model/labels'

export function StatusChip({ status }: { status: string }) {
	return (
		<Chip color={statusColor(status)} variant='soft' size='sm'>
			{statusLabel(status)}
		</Chip>
	)
}

export function PaymentChip({ type }: { type: string }) {
	return (
		<Chip variant='soft' size='sm'>
			{paymentLabel(type)}
		</Chip>
	)
}
