import { connection } from 'next/server'
import { getDeliveryTowns, getMenu } from '@/entities/dish'
import { getWorkingHours } from '@/shared/api'
import { serverEnv } from '@/shared/config'
import { CheckoutForm } from './checkout-form'
import { CheckoutHeader } from './checkout-header'
import { TitleRow } from './title-row'

export async function CheckoutPage({ paymentFailed = false }: { paymentFailed?: boolean }) {
	// Rendered per request: a build must not need the backend, and a backend failure must reach error.tsx.
	await connection()
	const [menu, towns, hours] = await Promise.all([getMenu(), getDeliveryTowns(), getWorkingHours()])

	return (
		<>
			<CheckoutHeader step={2} />
			<main>
				<CheckoutForm
					dishes={menu.dishes}
					towns={towns}
					hours={hours}
					paymentFailed={paymentFailed}
					suggestions={Boolean(serverEnv().DADATA_API_KEY)}
					intro={<TitleRow step={2} title='Оформление' />}
				/>
			</main>
		</>
	)
}
