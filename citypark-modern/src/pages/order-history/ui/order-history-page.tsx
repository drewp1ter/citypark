import { PaymentRetry } from '@/features/payment-retry'
import { redirect } from 'next/navigation'
import { CustomerApiError } from '@/shared/customer'
import { formatDate } from '@/shared/lib'
import { AppLink, cardClass } from '@/shared/ui'
import { getMyOrders, formatOrderPrice, orderPayable, orderStatusLabel } from '@/entities/order'

export async function OrderHistoryPage({ searchParams }: { searchParams: Promise<{ page?: string }> }) {
 const raw = (await searchParams).page ?? '1'
 const page = /^[1-9]\d*$/.test(raw) && Number.isSafeInteger(Number(raw)) ? Number(raw) : 1
 let result
 try { result = await getMyOrders(page) }
 catch (error) {
  if (error instanceof CustomerApiError && error.status === 401) redirect('/account/sign-in?returnTo=%2Faccount%2Forders')
  result = null
 }
 if (result === undefined) redirect('/account/sign-in?returnTo=%2Faccount%2Forders')
 const pages = result ? Math.ceil(result.count / 10) : 0
 return <main className='max-w-page mx-auto w-full px-3 py-10 md:px-6 md:py-16'><div className='mx-auto flex max-w-3xl flex-col gap-6'>
  <h1 className='heading text-4xl'>История заказов</h1>
  <AppLink href='/account' className='min-h-11 self-start py-2 underline'>Личный кабинет</AppLink>
  {result === null ? <p role='alert'>Не удалось загрузить заказы. Попробуйте позже</p> : result.count === 0 ? <p>У вас пока нет заказов</p> : <>
   <ul className='flex flex-col gap-3'>{result.orders.map(order => <li key={order.id} className={cardClass('surface', 'gap-2 rounded-lg p-5')}>
    <AppLink href={`/account/orders/${order.id}`} className='min-h-11 font-semibold underline'>Заказ № {order.id}</AppLink>
    <span>{formatDate(order.createdAt)} · {orderStatusLabel(order.status)}</span>
    <span>Товары: {formatOrderPrice(order, order.total)} · Доставка: {formatOrderPrice(order, order.deliveryPrice)} · Итого: {formatOrderPrice(order, orderPayable(order))}</span>
    <PaymentRetry orderId={order.id} status={order.status} eligibility={order.paymentRetry} surface='web' />
   </li>)}</ul>
   <nav aria-label='Страницы заказов' className='flex flex-wrap gap-3'>
    {page > 1 && <AppLink className='min-h-11 py-2 underline' href={`/account/orders?page=${page - 1}`}>Предыдущая</AppLink>}
    <span className='min-h-11 py-2'>Страница {page} из {pages}</span>
    {page < pages && <AppLink className='min-h-11 py-2 underline' href={`/account/orders?page=${page + 1}`}>Следующая</AppLink>}
   </nav>
  </>}
 </div></main>
}
