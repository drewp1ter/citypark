'use client'

import { useMemo } from 'react'
import { useCartLines, useCartReady } from '@/entities/cart'
import { priceLines, type Dish, type PricedCart } from '@/entities/dish'

/** The cart joined with the menu (names, photos, prices). `ready` is false until the browser has read the cart. */
export function usePricedCart(dishes: readonly Dish[]): PricedCart & { ready: boolean } {
	const lines = useCartLines()
	const ready = useCartReady()
	const priced = useMemo(() => priceLines(lines, dishes), [lines, dishes])
	return { ...priced, ready }
}
