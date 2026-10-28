import { connection } from 'next/server'
import { getDeliveryTowns, getMenu } from '@/entities/dish'
import { CheckoutForm } from './checkout-form'
import { CheckoutHeader } from './checkout-header'
import { TitleRow } from './title-row'

export async function CheckoutPage({ paymentFailed = false }: { paymentFailed?: boolean }) {
	// Rendered per request: a build must not need the backend, and a backend failure must reach error.tsx.
	await connection()
	const [menu, towns] = await Promise.all([getMenu(), getDeliveryTowns()])

	return (
		<>
			<CheckoutHeader step={2} />
			<main>
				<CheckoutForm
					dishes={menu.dishes}
					towns={towns}
					paymentFailed={paymentFailed}
					intro={<TitleRow step={2} title='Оформление' />}
				/>
			</main>
		</>
	)
}
