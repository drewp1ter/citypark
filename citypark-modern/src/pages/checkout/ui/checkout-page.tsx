import { connection } from 'next/server'
import { getMenu } from '@/entities/dish'
import { CheckoutForm } from './checkout-form'
import { CheckoutHeader } from './checkout-header'
import { TitleRow } from './title-row'

export async function CheckoutPage() {
	// Rendered per request: a build must not need the backend, and a backend failure must reach error.tsx.
	await connection()
	const menu = await getMenu()

	return (
		<>
			<CheckoutHeader step={2} />
			<main>
				<CheckoutForm dishes={menu.dishes} intro={<TitleRow step={2} title='Оформление' />} />
			</main>
		</>
	)
}
