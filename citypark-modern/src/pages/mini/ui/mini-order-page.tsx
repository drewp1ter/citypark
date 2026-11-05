import { PaymentRetry, PaymentReturnStatus } from '@/features/payment-retry'
import { notFound, redirect } from 'next/navigation'
import { getMyOrder, OrderBreakdown, orderStatusLabel } from '@/entities/order'
import { CustomerApiError } from '@/shared/customer'
import { formatDate } from '@/shared/lib'
import { AppLink, cardClass } from '@/shared/ui'

export async function MiniOrderPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ payment?: string }> }) {
 const payment = (await searchParams).payment
 const raw = (await params).id
 if (!/^[1-9]\d*$/.test(raw) || !Number.isSafeInteger(Number(raw))) notFound()
 let order
 try { order = await getMyOrder(Number(raw), 'mini') }
 catch (error) {
  if (error instanceof CustomerApiError && error.status === 401) redirect('/mini')
  if (error instanceof CustomerApiError && error.status === 404) notFound()
  return <main className='mx-auto max-w-3xl px-4 py-6'><h1 className='heading text-3xl'>Заказ недоступен</h1><p role='alert'>Не удалось загрузить заказ. Попробуйте позже</p></main>
 }
 if (!order) redirect('/mini')
 return <main className='mx-auto flex max-w-3xl flex-col gap-5 px-4 py-6 pb-[var(--mini-bottom,env(safe-area-inset-bottom))]'>
  <h1 className='heading text-3xl'>Заказ № {order.id}</h1>
  <AppLink href='/mini/orders' className='min-h-11 py-2 underline'>Все заказы</AppLink>
  <div className={cardClass('surface', 'gap-3 rounded-lg p-5')}>
   <p>Дата: {formatDate(order.createdAt)}</p><p>Статус: {orderStatusLabel(order.status)}</p>
   <p>Получатель: {order.customer.name}, {order.customer.phone}</p>
   <p>{order.address ? `Доставка: ${order.address.city}, ${order.address.address}` : 'Самовывоз'}</p>
   {order.deliveryTime && <p>Время доставки: {order.deliveryTime}</p>}
   {order.comment && <p>Комментарий: {order.comment}</p>}
   <h2 className='font-display text-xl'>Состав заказа</h2>
   <OrderBreakdown order={order} />
   <PaymentReturnStatus status={order.status} payment={payment} />
   <PaymentRetry orderId={order.id} status={order.status} eligibility={order.paymentRetry} surface='mini' />
  </div>
 </main>
}
