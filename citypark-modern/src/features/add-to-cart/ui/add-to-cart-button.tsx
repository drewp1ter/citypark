'use client'

import { cartActions, useCartQuantity } from '@/entities/cart'
import type { Dish } from '@/entities/dish'
import { formatPrice } from '@/shared/lib'
import { IconButton, MinusIcon, PlusIcon } from '@/shared/ui'

type AddToCartButtonProps = {
	dish: Pick<Dish, 'id' | 'name' | 'variants' | 'available'>
	/** Size to add; the first one by default. */
	variantId?: string
	/**
	 * "icon": the round lime "+" that becomes a "− 2 +" stepper (home page, desktop menu).
	 * "pill": a full-width pill with the price, "550 ₽ +", that becomes "− 2 · 1 100 ₽ +" (mobile menu).
	 */
	layout?: 'icon' | 'pill'
}

const stepButton = 'inline-flex size-11 shrink-0 cursor-pointer items-center justify-center text-lg'

/**
 * Adds a dish to the cart. It is disabled while the dish is on the stop-list or the chosen size has no price yet.
 */
export function AddToCartButton({ dish, variantId, layout = 'icon' }: AddToCartButtonProps) {
	const variant = dish.variants.find((v) => v.id === variantId) ?? dish.variants[0]
	const key = { dishId: dish.id, variantId: variant.id }
	const quantity = useCartQuantity(key)
	const canOrder = dish.available && variant.price !== null
	const addLabel = `Добавить в корзину: ${dish.name.toLocaleLowerCase('ru')}`

	const minus = (
		<button
			type='button'
			aria-label='Убрать порцию'
			className={stepButton}
			onClick={() => cartActions.setQuantity(key, quantity - 1)}
		>
			<MinusIcon />
		</button>
	)
	const plus = (
		<button type='button' aria-label='Добавить порцию' className={stepButton} onClick={() => cartActions.add(key)}>
			<PlusIcon />
		</button>
	)

	if (layout === 'pill') {
		if (quantity === 0) {
			return (
				<button
					type='button'
					disabled={!canOrder}
					onClick={() => cartActions.add(key)}
					className='rounded-pill bg-accent text-on-accent min-h-11 w-full cursor-pointer text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-50'
				>
					{/* The visible text ("550 ₽ +") stays inside the accessible name: "Добавить в корзину: салат, 550 ₽ +". */}
					<span className='sr-only'>{addLabel}, </span>
					{variant.price === null ? 'Цена уточняется' : `${formatPrice(variant.price)} +`}
				</button>
			)
		}
		return (
			<span className='rounded-pill bg-inverse-bg text-inverse-text flex items-center justify-between'>
				{minus}
				<span aria-live='polite' className='text-sm font-semibold'>
					{quantity} · {formatPrice((variant.price ?? 0) * quantity)}
				</span>
				{plus}
			</span>
		)
	}

	if (quantity === 0) {
		return (
			<IconButton variant='accent' aria-label={addLabel} disabled={!canOrder} onClick={() => cartActions.add(key)}>
				<PlusIcon width={20} height={20} />
			</IconButton>
		)
	}

	return (
		<span className='rounded-pill bg-inverse-bg text-inverse-text inline-flex shrink-0 items-center'>
			{minus}
			<span aria-live='polite' className='min-w-3.5 text-center font-semibold'>
				{quantity}
			</span>
			{plus}
		</span>
	)
}
