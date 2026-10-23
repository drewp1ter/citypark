'use client'

import type { MouseEvent, RefObject } from 'react'
import { quoteDelivery, type Dish } from '@/entities/dish'
import { useSiteSettings } from '@/shared/config'
import { formatPrice, plural } from '@/shared/lib'
import { AppLink, Button, CloseIcon, IconButton } from '@/shared/ui'
import { CartLine } from './cart-line'
import { usePricedCart } from './use-priced-cart'

type CartSheetProps = {
	dishes: readonly Dish[]
	dialog: RefObject<HTMLDialogElement | null>
	onClose: () => void
}

/**
 * The cart as a bottom sheet (below 1024 px), opened from the CartBar.
 * It is a native modal <dialog>, like MobileMenu: Esc, focus trap and an inert page come from the browser.
 */
export function CartSheet({ dishes, dialog, onClose }: CartSheetProps) {
	const { items, count, subtotal } = usePricedCart(dishes)
	const { restaurant } = useSiteSettings().phones
	const quote = quoteDelivery({ method: 'delivery', zone: 'city', subtotal })

	const closeOnBackdropOrLink = (event: MouseEvent<HTMLDialogElement>) => {
		const target = event.target as HTMLElement
		if (target === dialog.current || target.closest('a')) dialog.current?.close()
	}

	return (
		<dialog
			ref={dialog}
			aria-labelledby='cart-sheet-title'
			onClick={closeOnBackdropOrLink}
			onClose={onClose}
			className='bg-inverse-bg text-inverse-text backdrop:bg-scrim fixed inset-x-0 top-auto bottom-0 m-0 max-h-[85dvh] w-full max-w-none overflow-y-auto rounded-t-xl p-5 pb-6 lg:hidden'
		>
			<div className='flex items-center justify-between gap-3'>
				<h2 id='cart-sheet-title' className='heading text-2xl'>
					Корзина
				</h2>
				<span className='text-inverse-muted ml-auto text-sm'>
					{count} {plural(count, ['блюдо', 'блюда', 'блюд'])}
				</span>
				<IconButton variant='inverse' aria-label='Закрыть корзину' onClick={() => dialog.current?.close()}>
					<CloseIcon />
				</IconButton>
			</div>

			<ul className='mt-2'>
				{items.map((item) => (
					<CartLine key={`${item.dish.id}:${item.variant.id}`} item={item} />
				))}
			</ul>

			<div className='mt-3.5 flex justify-between text-[15px]'>
				<span className='text-inverse-muted'>Доставка</span>
				<span className='text-accent font-semibold'>{quote.label}</span>
			</div>
			<div className='mt-3.5 flex items-baseline justify-between'>
				<span>Итого</span>
				<span className='font-display text-[30px] font-medium'>{formatPrice(subtotal)}</span>
			</div>
			<Button href='/checkout' className='mt-3.5 min-h-[54px] w-full'>
				Оформить заказ
			</Button>
			<p className='text-inverse-muted mt-3.5 text-center text-[13px]'>
				или{' '}
				<AppLink href={restaurant.href} className='tap-area text-inverse-text'>
					{restaurant.display}
				</AppLink>
			</p>
		</dialog>
	)
}
