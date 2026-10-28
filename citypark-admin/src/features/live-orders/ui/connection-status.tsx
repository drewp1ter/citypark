import { Chip } from '@heroui/react'
import type { LiveStatus } from '../model/socket'

const view: Record<LiveStatus, { text: string; color: 'success' | 'warning' | 'danger' }> = {
	online: { text: 'Онлайн', color: 'success' },
	connecting: { text: 'Подключение…', color: 'warning' },
	offline: { text: 'Нет связи', color: 'danger' },
}

export function ConnectionStatus({ status }: { status: LiveStatus }) {
	const { text, color } = view[status]
	return (
		<span aria-live='polite'>
			<Chip color={color} variant='soft'>
				{text}
			</Chip>
		</span>
	)
}
