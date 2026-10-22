'use client'

import { useCartCount } from '@/entities/cart'
import { CartIcon, IconButton } from '@/shared/ui'

/** Header cart button with the number of portions in the cart. */
export function CartButton() {
	const count = useCartCount()

	return (
		<IconButton href='/menu' aria-label={count > 0 ? `Корзина, блюд: ${count}` : 'Корзина'} className='relative'>
			<CartIcon />
			{count > 0 && (
				<span
					aria-hidden='true'
					className='bg-accent text-on-accent absolute -top-1 -right-1 grid h-5 min-w-5 place-items-center rounded-full px-1 text-xs font-semibold'
				>
					{count}
				</span>
			)}
		</IconButton>
	)
}
