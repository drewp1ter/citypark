import type { Order } from '@/entities/order'

export type RetryState = { error?: string }
export type PaymentRetryProps = { orderId: number; status: string; eligibility: Order['paymentRetry']; surface: 'web' | 'mini' }
