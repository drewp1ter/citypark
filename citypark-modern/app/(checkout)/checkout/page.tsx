import type { Metadata } from 'next'
import { CheckoutPage } from '@/pages/checkout'

export const metadata: Metadata = { title: 'Оформление заказа — Сити Парк' }

export default async function Page({ searchParams }: PageProps<'/checkout'>) {
	const { payment, MNT_TRANSACTION_ID } = await searchParams
	return <CheckoutPage paymentFailed={payment === 'failed'} returnedOrderId={typeof MNT_TRANSACTION_ID === 'string' ? MNT_TRANSACTION_ID : undefined} />
}
