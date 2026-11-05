import { PaymentRetry } from '@/features/payment-retry'
import { redirect } from 'next/navigation'
import { getMyOrders, formatOrderPrice, orderPayable, orderStatusLabel } from '@/entities/order'
import { CustomerApiError } from '@/shared/customer'
import { formatDate } from '@/shared/lib'
import { AppLink, cardClass } from '@/shared/ui'

export async function MiniHistoryPage({ searchParams }: { searchParams: Promise<{ page?: string }> }) {
 const raw = (await searchParams).page ?? '1'
 const page = /^[1-9]\d*$/.test(raw) && Number.isSafeInteger(Number(raw)) ? Number(raw) : 1
 let result
 try { result = await getMyOrders(page, 'mini') }
 catch (error) {
  if (error instanceof CustomerApiError && error.status === 401) redirect('/mini')
  result = null
 }
 if (result === undefined) redirect('/mini')
 const pages = result ? Math.ceil(result.count / 10) : 0
 return <main className='mx-auto flex max-w-3xl flex-col gap-5 px-4 py-6 pb-[var(--mini-bottom,env(safe-area-inset-bottom))]'>
  <h1 className='heading text-3xl'>История заказов</h1>
  <AppLink href='/mini/account' className='min-h-11 py-2 underline'>Аккаунт</AppLink>
  {result === null ? <p role='alert'>Не удалось загрузить заказы. Попробуйте позже</p> : result.count === 0 ? <p>У вас пока нет заказов</p> : <>
   <ul className='flex flex-col gap-3'>{result.orders.map(order => <li key={order.id} className={cardClass('surface', 'gap-2 rounded-lg p-5')}>
    <AppLink href={`/mini/orders/${order.id}`} className='min-h-11 font-semibold underline'>Заказ № {order.id}</AppLink>
    <span>{formatDate(order.createdAt)} · {orderStatusLabel(order.status)}</span>
    <span>Итого: {formatOrderPrice(order, orderPayable(order))}</span>
    <PaymentRetry orderId={order.id} status={order.status} eligibility={order.paymentRetry} surface='mini' />
   </li>)}</ul>
   <nav aria-label='Страницы заказов' className='flex flex-wrap gap-3'>
    {page > 1 && <AppLink className='min-h-11 py-2 underline' href={`/mini/orders?page=${page - 1}`}>Предыдущая</AppLink>}
    <span className='min-h-11 py-2'>Страница {page} из {pages}</span>
    {page < pages && <AppLink className='min-h-11 py-2 underline' href={`/mini/orders?page=${page + 1}`}>Следующая</AppLink>}
   </nav>
  </>}
 </main>
}
