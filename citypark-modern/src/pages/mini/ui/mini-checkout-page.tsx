import { loadAddressBook } from '@/entities/address/index.server'
import { connection } from 'next/server'
import { getDeliveryTowns, getMenu } from '@/entities/dish'
import { getMiniProfile } from '@/features/telegram-launch'
import { getSiteSettings, getWorkingHours } from '@/shared/api'
import { serverEnv } from '@/shared/config'
import { AppLink } from '@/shared/ui'
import { MiniPaymentReturnPage } from './mini-payment-return-page'
import { CheckoutForm } from '@/widgets/food-ordering'

export async function MiniCheckoutPage({
	searchParams,
}: { searchParams?: Promise<{ payment?: string; MNT_TRANSACTION_ID?: string }> } = {}) {
	await connection()
	if (!(await getMiniProfile())) return null
	if (searchParams && (await searchParams).payment === 'failed')
		return <MiniPaymentReturnPage searchParams={searchParams} failed />
	const [menu, towns, hours, settings, addressBook] = await Promise.all([
		getMenu(),
		getDeliveryTowns(),
		getWorkingHours(),
		getSiteSettings(),
		loadAddressBook('mini'),
	])
	return (
		<div className='min-h-[var(--mini-height,100dvh)] pt-[var(--mini-top,env(safe-area-inset-top))] pr-[var(--mini-right,env(safe-area-inset-right))] pb-[var(--mini-bottom,env(safe-area-inset-bottom))] pl-[var(--mini-left,env(safe-area-inset-left))]'>
			<header className='border-border flex min-h-14 items-center gap-3 border-b px-4'>
				<AppLink href='/mini' className='tap-area rounded-pill px-3 py-2'>
					← Меню
				</AppLink>
				<span className='font-display font-semibold'>Сити Парк</span>
			</header>
			<main id='content'>
				<CheckoutForm
					mini
					addressBook={addressBook}
					dishes={menu.dishes}
					towns={towns}
					hours={hours}
					promoCodesEnabled={settings.promoCodesEnabled}
					paymentFailed={false}
					suggestions={Boolean(serverEnv().DADATA_API_KEY)}
					locationEnabled={Boolean(serverEnv().DADATA_API_KEY)}
					intro={<h1 className='heading mb-4 text-3xl'>Оформление заказа</h1>}
				/>
			</main>
		</div>
	)
}
