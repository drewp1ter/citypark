'use client'

import { useRef, useState } from 'react'
import type { Dish } from '@/entities/dish'
import { formatPrice, plural } from '@/shared/lib'
import { ChevronDownIcon } from '@/shared/ui'
import { CartSheet } from './cart-sheet'
import { usePricedCart } from './use-priced-cart'

/** Sticky bar at the bottom of the menu page below 1024 px: number of dishes, total; it opens the cart sheet. */
export function CartBar({ dishes }: { dishes: readonly Dish[] }) {
	const { count, subtotal } = usePricedCart(dishes)
	const [open, setOpen] = useState(false)
	const dialog = useRef<HTMLDialogElement>(null)
	// At 0 dishes the bar unmounts and takes the open sheet with it.
	if (count === 0) return null

	return (
		<>
			<nav
				aria-label='Корзина'
				className='to-bg sticky bottom-0 z-10 bg-linear-to-b from-transparent to-35% px-3 pt-2.5 pb-3 lg:hidden'
			>
				<button
					type='button'
					aria-haspopup='dialog'
					aria-expanded={open}
					onClick={() => {
						dialog.current?.showModal()
						setOpen(true)
					}}
					className='rounded-pill bg-inverse-bg text-inverse-text flex min-h-[58px] w-full cursor-pointer items-center justify-between pr-2 pl-[22px] font-semibold'
				>
					<span>
						Корзина · {count} {plural(count, ['блюдо', 'блюда', 'блюд'])}
					</span>
					<span className='rounded-pill bg-accent text-on-accent flex items-center gap-1.5 px-4 py-2.5'>
						{formatPrice(subtotal)}
						<ChevronDownIcon className='rotate-180' aria-hidden />
					</span>
				</button>
			</nav>
			<CartSheet dishes={dishes} dialog={dialog} onClose={() => setOpen(false)} />
		</>
	)
}
