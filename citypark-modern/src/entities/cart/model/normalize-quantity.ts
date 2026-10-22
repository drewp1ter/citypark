/**
 * Turns a quantity that comes from the UI into the quantity the cart will store.
 * The reducer removes the line when this returns 0 or less.
 *
 * TODO(you): decide the rules. Questions to answer:
 *  - What happens to 0 and negative numbers? (remove the line? clamp to 1?)
 *  - Is there a maximum per line? A cap protects the kitchen from a "999 portions" typo,
 *    but a silent cap can confuse a real big banquet order. (Choose a number, or none.)
 *  - What about 2.5, NaN or Infinity? (they can only come from a bug or a hacked localStorage value)
 *
 * The placeholder below only rounds down and turns NaN / Infinity into 0, with no maximum.
 */
export function normalizeQuantity(quantity: number): number {
	return Number.isFinite(quantity) ? Math.trunc(quantity) : 0
}
