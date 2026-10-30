import 'server-only'
import { customerRequest, getCustomerSessionToken, getMiniSessionToken } from '@/shared/customer'
import type { Order } from '../model/types'

export type OrderSurface = 'web' | 'mini'

export async function getMyOrders(page: number, surface: OrderSurface = 'web') {
 const token = surface === 'mini' ? await getMiniSessionToken() : await getCustomerSessionToken()
 if (!token) return undefined
 return customerRequest<{ orders: Order[]; count: number }>(`/orders/my?page=${page}&pageSize=10`, { headers: { Authorization: `Bearer ${token}` } })
}

export async function getMyOrder(id: number, surface: OrderSurface = 'web') {
 const token = surface === 'mini' ? await getMiniSessionToken() : await getCustomerSessionToken()
 if (!token) return undefined
 return customerRequest<Order>(`/orders/${id}`, { headers: { Authorization: `Bearer ${token}` } })
}

export function orderStatusLabel(status: string): string {
 return ({ new: 'Новый', pending: 'Ожидает оплаты', paid: 'Оплачен', canceled: 'Отменён', confirmed: 'Подтверждён' } as Record<string, string>)[status] ?? 'Статус неизвестен'
}
