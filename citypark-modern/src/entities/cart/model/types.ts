/**
 * A cart line holds only ids and a quantity, never a name or a price.
 * Prices are looked up in the menu when the cart is shown or an order is checked.
 */
export type CartLine = {
  dishId: string;
  variantId: string;
  quantity: number;
};

/** Identifies a line: one dish in one size. */
export type CartLineKey = Pick<CartLine, "dishId" | "variantId">;
