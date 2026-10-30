import { notFound, redirect } from 'next/navigation'
import { CustomerApiError } from '@/shared/customer'
import { formatDate, formatPrice } from '@/shared/lib'
import { AppLink, cardClass } from '@/shared/ui'
import { getMyOrder, orderStatusLabel } from '../api/orders'

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
   <ul className='flex flex-col gap-2'>{order.items.map((item, index) => <li key={`${item.productId}-${index}`} className='flex justify-between gap-3'><span>{item.name} × {item.amount}</span><span>{formatPrice(item.offeredPrice * item.amount)}</span></li>)}</ul>
   <p>Товары: {formatPrice(order.total)}</p><p>Доставка: {formatPrice(order.deliveryPrice)}</p>
   <p className='font-semibold'>Итого: {formatPrice(order.total + order.deliveryPrice)}</p>
  </div>
 </div></main>
}
