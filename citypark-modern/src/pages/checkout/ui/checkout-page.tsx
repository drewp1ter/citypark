import { connection } from 'next/server'
import { redirect } from 'next/navigation'
import { getPendingPaymentContext } from '@/shared/customer'
import { getDeliveryTowns, getMenu } from '@/entities/dish'
import { getSiteSettings, getWorkingHours } from '@/shared/api'
import { serverEnv } from '@/shared/config'
import { CheckoutForm } from '@/widgets/food-ordering'
import { CheckoutHeader } from './checkout-header'
import { TitleRow } from './title-row'

export async function CheckoutPage({ paymentFailed = false, returnedOrderId }: { paymentFailed?: boolean; returnedOrderId?: string }) {
	if (paymentFailed) {
		const context = await getPendingPaymentContext(returnedOrderId)
		if (context?.surface === 'mini') redirect(`/mini/checkout?payment=failed&MNT_TRANSACTION_ID=${context.orderId}`)
	}
	// Rendered per request: a build must not need the backend, and a backend failure must reach error.tsx.
	await connection()
	const [menu, towns, hours, settings] = await Promise.all([getMenu(), getDeliveryTowns(), getWorkingHours(), getSiteSettings()])

	return (
		<>
			<CheckoutHeader step={2} />
			<main>
				<CheckoutForm
					dishes={menu.dishes}
					towns={towns}
					hours={hours}
					promoCodesEnabled={settings.promoCodesEnabled}
					paymentFailed={paymentFailed}
					suggestions={Boolean(serverEnv().DADATA_API_KEY)}
					intro={<TitleRow step={2} title='Оформление' />}
				/>
			</main>
		</>
	)
}
