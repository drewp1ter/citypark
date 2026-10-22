import { normalizeQuantity } from './normalize-quantity'
import type { CartLine, CartLineKey } from './types'

export type CartAction =
	| ({ type: 'add'; quantity?: number } & CartLineKey)
	| ({ type: 'setQuantity'; quantity: number } & CartLineKey)
	| ({ type: 'remove' } & CartLineKey)
	| { type: 'clear' }

const isSameLine = (a: CartLineKey, b: CartLineKey) => a.dishId === b.dishId && a.variantId === b.variantId

/** Sets the quantity of one line. Quantity 0 (after normalizing) removes it; new lines go to the end. */
function withQuantity(lines: readonly CartLine[], key: CartLineKey, quantity: number): readonly CartLine[] {
	const next = normalizeQuantity(quantity)
	if (next <= 0) return lines.filter((line) => !isSameLine(line, key))

	if (lines.some((line) => isSameLine(line, key))) {
		return lines.map((line) => (isSameLine(line, key) ? { ...line, quantity: next } : line))
	}
	return [...lines, { dishId: key.dishId, variantId: key.variantId, quantity: next }]
}

export function cartReducer(lines: readonly CartLine[], action: CartAction): readonly CartLine[] {
	switch (action.type) {
		case 'add': {
			const current = lines.find((line) => isSameLine(line, action))?.quantity ?? 0
			return withQuantity(lines, action, current + (action.quantity ?? 1))
		}
		case 'setQuantity':
			return withQuantity(lines, action, action.quantity)
		case 'remove':
			return withQuantity(lines, action, 0)
		case 'clear':
			return []
	}
}
