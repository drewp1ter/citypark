import { getSiteSettings, getWorkingHours } from '@/shared/api'
import { formatHours } from '@/shared/lib'
import { Button, cardClass } from '@/shared/ui'
import { CheckoutHeader } from './checkout-header'
import { ClearCart } from './clear-cart'
import { TitleRow } from './title-row'

/** Step 3, "Готово". Moneta brings the customer here after a successful payment. */
export async function CheckoutDonePage() {
	const [{ phones }, hours] = await Promise.all([getSiteSettings(), getWorkingHours()])

	return (
		<>
			<CheckoutHeader step={3} />
			<main className='max-w-narrow-page mx-auto w-full px-3 pt-2 pb-6 md:px-6 md:pt-10 md:pb-10'>
				<ClearCart />
				<TitleRow step={3} title='Заказ принят' />
				<div className={cardClass('surface', 'items-start gap-3 rounded-lg p-6 md:rounded-xl md:p-10')}>
					<h2 className='heading text-2xl md:text-[32px]'>Спасибо за заказ!</h2>
					<p className='text-text-muted max-w-xl'>
						Оплата прошла, заказ принят. Мы перезвоним, если что-то нужно уточнить. Доставляем {formatHours(hours)}.
						Если что-то изменилось, позвоните:{' '}
						<span className='text-text whitespace-nowrap'>{phones.restaurant.display}</span>.
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
