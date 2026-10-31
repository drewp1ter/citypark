import { formatPrice } from '@/shared/lib'
import type { Order } from './types'

const money = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 })

export function formatOrderPrice(order: Order, value: number): string {
	return order.promoCode ? `${money.format(value)}\u00a0₽` : formatPrice(value)
}

export function orderPayable(order: Order): number {
	return (Math.round(order.total * 100) + Math.round(order.deliveryPrice * 100)) / 100
}
