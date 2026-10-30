import 'server-only'
import { customerRequest, getCustomerSessionToken } from '@/shared/customer'

export type Order = {
 id: number; createdAt: string; status: string; comment: string; deliveryTime: string;
 total: number; deliveryPrice: number;
 customer: { id: number; name: string; phone: string };
 address: { id: number; city: string; address: string } | null;
 items: { productId: number; name: string; categoryName: string; amount: number; offeredPrice: number }[];
}

export async function getMyOrders(page: number) {
 const token = await getCustomerSessionToken()
 if (!token) return undefined
 return customerRequest<{ orders: Order[]; count: number }>(`/orders/my?page=${page}&pageSize=10`, { headers: { Authorization: `Bearer ${token}` } })
}

export async function getMyOrder(id: number) {
 const token = await getCustomerSessionToken()
 if (!token) return undefined
 return customerRequest<Order>(`/orders/${id}`, { headers: { Authorization: `Bearer ${token}` } })
}

export function orderStatusLabel(status: string): string {
 return ({ new: 'Новый', pending: 'Ожидает оплаты', paid: 'Оплачен', canceled: 'Отменён', confirmed: 'Подтверждён' } as Record<string, string>)[status] ?? 'Статус неизвестен'
}
