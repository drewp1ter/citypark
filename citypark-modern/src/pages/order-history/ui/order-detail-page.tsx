import { notFound, redirect } from 'next/navigation'
import { CustomerApiError } from '@/shared/customer'
import { formatDate } from '@/shared/lib'
import { AppLink, cardClass } from '@/shared/ui'
import { getMyOrder, OrderBreakdown, orderStatusLabel } from '@/entities/order'

export async function OrderDetailPage({ params }: { params: Promise<{ id: string }> }) {
 const raw = (await params).id
 if (!/^[1-9]\d*$/.test(raw) || !Number.isSafeInteger(Number(raw))) notFound()
 let order
 try { order = await getMyOrder(Number(raw)) }
 catch (error) {
  if (error instanceof CustomerApiError && error.status === 401) redirect(`/account/sign-in?returnTo=${encodeURIComponent(`/account/orders/${raw}`)}`)
  if (error instanceof CustomerApiError && error.status === 404) notFound()
  return <main className='max-w-page mx-auto px-3 py-10'><h1 className='heading text-4xl'>Заказ недоступен</h1><p role='alert'>Не удалось загрузить заказ. Попробуйте позже</p></main>
 }
 if (!order) redirect(`/account/sign-in?returnTo=${encodeURIComponent(`/account/orders/${raw}`)}`)
 return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'><div className='mx-auto flex max-w-3xl flex-col gap-6'>
  <h1 className='heading text-4xl'>Заказ № {order.id}</h1>
  <AppLink href='/account/orders' className='min-h-11 self-start py-2 underline'>Все заказы</AppLink>
  <div className={cardClass('surface', 'gap-3 rounded-lg p-5 md:p-8')}>
   <p>Дата: {formatDate(order.createdAt)}</p><p>Статус: {orderStatusLabel(order.status)}</p>
   <p>Получатель: {order.customer.name}, {order.customer.phone}</p>
   <p>{order.address ? `Доставка: ${order.address.city}, ${order.address.address}` : 'Самовывоз'}</p>
   {order.deliveryTime && <p>Время доставки: {order.deliveryTime}</p>}
   {order.comment && <p>Комментарий: {order.comment}</p>}
   <h2 className='font-display text-xl'>Состав заказа</h2>
   <OrderBreakdown order={order} />
  </div>
 </div></main>
}
