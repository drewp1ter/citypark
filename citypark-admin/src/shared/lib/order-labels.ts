export const ORDER_STATUSES = ['new', 'pending', 'paid', 'confirmed', 'canceled'] as const
export type OrderStatus = (typeof ORDER_STATUSES)[number]

const statusLabels: Record<OrderStatus, string> = {
	new: 'Новый',
	pending: 'Ожидает оплаты',
	paid: 'Оплачен',
	canceled: 'Отменён',
	confirmed: 'Подтверждён',
}

export type ChipColor = 'default' | 'warning' | 'success' | 'danger'

const statusColors: Record<OrderStatus, ChipColor> = {
	new: 'default',
	pending: 'warning',
	paid: 'success',
	canceled: 'danger',
	confirmed: 'success',
}

export function isOrderStatus(value: unknown): value is OrderStatus {
	return typeof value === 'string' && (ORDER_STATUSES as readonly string[]).includes(value)
}

/** Russian name of a status; an unknown status is shown as it came from the API. */
export function statusLabel(status: string): string {
	return isOrderStatus(status) ? statusLabels[status] : status
}

export function statusColor(status: string): ChipColor {
	return isOrderStatus(status) ? statusColors[status] : 'default'
}
