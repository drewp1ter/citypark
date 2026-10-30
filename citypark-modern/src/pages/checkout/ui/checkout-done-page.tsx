import { getSiteSettings, getWorkingHours } from '@/shared/api'
import { formatHours } from '@/shared/lib'
import { Button, cardClass } from '@/shared/ui'
import { redirect } from 'next/navigation'
import { getPendingPaymentContext } from '@/shared/customer'
import { getMyOrder } from '@/entities/order'
import { CheckoutHeader } from './checkout-header'
import { ClearCart } from './clear-cart'
import { TitleRow } from './title-row'

/** Step 3, "Готово". Moneta brings the customer here after a successful payment. */
export async function CheckoutDonePage({ searchParams }: { searchParams: Promise<{ MNT_TRANSACTION_ID?: string }> }) {
	const context = await getPendingPaymentContext((await searchParams).MNT_TRANSACTION_ID)
	if (context?.surface === 'mini') redirect(`/mini/checkout/done?MNT_TRANSACTION_ID=${context.orderId}`)
	let status: 'paid' | 'pending' | 'failed' | 'unknown' = 'unknown'
	if (context?.surface === 'web') {
		try {
			const order = await getMyOrder(context.orderId)
			if (order) status = order.status === 'paid' || order.status === 'confirmed' ? 'paid' : order.status === 'canceled' ? 'failed' : 'pending'
		} catch { /* Keep the cart when order status cannot be verified. */ }
	}
	const [{ phones }, hours] = await Promise.all([getSiteSettings(), getWorkingHours()])

	return (
		<>
			<CheckoutHeader step={3} />
			<main className='max-w-narrow-page mx-auto w-full px-3 pt-2 pb-6 md:px-6 md:pt-10 md:pb-10'>
				{status === 'paid' && <ClearCart />}
				<TitleRow step={3} title={status === 'paid' ? 'Заказ принят' : 'Статус заказа'} />
				<div className={cardClass('surface', 'items-start gap-3 rounded-lg p-6 md:rounded-xl md:p-10')}>
					<h2 className='heading text-2xl md:text-[32px]'>{status === 'paid' ? 'Спасибо за заказ!' : 'Статус заказа'}</h2>
					<p className='text-text-muted max-w-xl'>
						{status === 'paid' ? `Заказ оплачен. Мы перезвоним, если что-то нужно уточнить. Доставляем ${formatHours(hours)}.` : status === 'pending' ? 'Платёж ещё обрабатывается. Корзина сохранена.' : status === 'failed' ? 'Оплата не завершена. Корзина сохранена.' : 'Не удалось проверить оплату. Корзина сохранена.'}{' '}
						Если нужна помощь, позвоните: <span className='text-text whitespace-nowrap'>{phones.restaurant.display}</span>.
					</p>
					<div className='mt-2 flex flex-wrap gap-2'>
						<Button href='/menu' size='xl' variant='inverse'>
							Вернуться в меню
						</Button>
						<Button href='/' size='xl' variant='outline'>
							На главную
						</Button>
					</div>
				</div>
			</main>
		</>
	)
}
