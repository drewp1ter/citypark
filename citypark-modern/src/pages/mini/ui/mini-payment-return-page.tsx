import { redirect } from 'next/navigation'
import { getMyOrder } from '@/entities/order'
import { getPendingPaymentContext } from '@/shared/customer'
import { AppLink, cardClass } from '@/shared/ui'
import { serverEnv } from '@/shared/config'
import { MiniClearCart } from './mini-clear-cart'
import { PendingPaymentRefresh } from './pending-payment-refresh'

export async function MiniPaymentReturnPage({ searchParams, failed = false }: { searchParams: Promise<{ MNT_TRANSACTION_ID?: string }>; failed?: boolean }) {
 const context = await getPendingPaymentContext((await searchParams).MNT_TRANSACTION_ID)
 if (context?.origin === 'history') redirect(`/${context.surface === 'mini' ? 'mini' : 'account'}/orders/${context.orderId}?payment=${failed ? 'failed' : 'returned'}`)
 const reopenUrl = serverEnv().TELEGRAM_MINI_APP_URL
 let status: 'paid' | 'pending' | 'failed' | 'missing' = 'missing'
 if (context?.surface === 'mini') {
  try {
   const order = await getMyOrder(context.orderId, 'mini')
   if (order) status = order.status === 'paid' || order.status === 'confirmed' ? 'paid' : order.status === 'canceled' || failed ? 'failed' : 'pending'
  } catch { /* A lost session or backend failure must not clear the cart. */ }
 }
 return <main className='mx-auto flex max-w-3xl flex-col gap-5 px-4 py-8 pb-[var(--mini-bottom,env(safe-area-inset-bottom))]'>
  <h1 className='heading text-3xl'>Результат оплаты</h1>
  <div className={cardClass('surface', 'gap-4 rounded-lg p-5')}>
   {status === 'paid' && <><p role='status'>Заказ оплачен</p><MiniClearCart /></>}
   {status === 'pending' && <><p role='status'>Платёж ещё обрабатывается</p><PendingPaymentRefresh /></>}
   {status === 'failed' && <p role='status'>Оплата не завершена</p>}
   {status === 'missing' && <p role='status'>Не удалось проверить заказ. Откройте приложение заново через Telegram.</p>}
   <div className='flex flex-wrap gap-3'>
    {context?.surface === 'mini' && status !== 'missing' && <AppLink href={`/mini/orders/${context.orderId}`} className='min-h-11 py-2 underline'>К заказу</AppLink>}
    <AppLink href='/mini' className='min-h-11 py-2 underline'>В меню</AppLink>
    {reopenUrl && <a href={reopenUrl} className='min-h-11 py-2 underline'>Открыть через Telegram</a>}
   </div>
  </div>
 </main>
}
