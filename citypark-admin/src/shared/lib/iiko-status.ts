import type { components } from '../api/schema'

type IikoStatus = components['schemas']['Order']['iikoStatus']

const labels: Record<NonNullable<IikoStatus>, string> = {
	awaiting_approval: 'Ожидает подтверждения',
	queued: 'В очереди на отправку',
	sending: 'Отправляется в iiko',
	created: 'Создан в iiko',
	failed: 'Ошибка отправки в iiko',
	review: 'Проверить в iiko',
}

export function iikoStatusLabel(status: IikoStatus) {
	return status === undefined ? undefined : labels[status]
}

export function isSendingToIiko(status: IikoStatus) {
	return status === 'queued' || status === 'sending'
}
