'use client'

import { useMemo } from 'react'
import { useCartLines } from '@/entities/cart'
import { priceLines, type Dish } from '@/entities/dish'
import { AppLink } from '@/shared/ui'

/** Menu header cart link with the number of available portions. */
export function CartPill({ dishes }: { dishes: readonly Dish[] }) {
	const lines = useCartLines()
	const { count } = useMemo(() => priceLines(lines, dishes), [lines, dishes])

	return (
		<AppLink
			href='#cart'
			aria-label={count > 0 ? `Корзина, блюд: ${count}` : 'Корзина'}
			className='rounded-pill bg-inverse-bg text-inverse-text inline-flex min-h-11 items-center gap-2.5 px-5 font-semibold'
		>
			Корзина
			{count > 0 && (
				<span aria-hidden='true' className='rounded-pill bg-accent text-on-accent px-2.5 py-1 text-sm'>
					{count}
				</span>
			)}
		</AppLink>
	)
}
