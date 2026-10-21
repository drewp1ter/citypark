"use client";

import { useSyncExternalStore } from "react";
import { dispatchCart, getCartSnapshot, getServerCartSnapshot, subscribeToCart } from "./cart-store";
import type { CartLine, CartLineKey } from "./types";

/** All lines, in the order they were added. */
export function useCartLines(): readonly CartLine[] {
  return useSyncExternalStore(subscribeToCart, getCartSnapshot, getServerCartSnapshot);
}

const subscribeNever = () => () => {};

/** False on the server and during hydration, true afterwards: the cart is only known in the browser. */
export function useCartReady(): boolean {
  return useSyncExternalStore(subscribeNever, () => true, () => false);
}

/** Number of portions in the cart (the badge on the cart button). */
export function useCartCount(): number {
  return useSyncExternalStore(
    subscribeToCart,
    () => getCartSnapshot().reduce((sum, line) => sum + line.quantity, 0),
    () => 0,
  );
}

/** Quantity of one dish in one size; 0 when it is not in the cart. A dish card re-renders only when its own quantity changes. */
export function useCartQuantity({ dishId, variantId }: CartLineKey): number {
  return useSyncExternalStore(
    subscribeToCart,
    () => getCartSnapshot().find((line) => line.dishId === dishId && line.variantId === variantId)?.quantity ?? 0,
    () => 0,
  );
}

export const cartActions = {
  add: (key: CartLineKey, quantity = 1) => dispatchCart({ type: "add", ...key, quantity }),
  setQuantity: (key: CartLineKey, quantity: number) => dispatchCart({ type: "setQuantity", ...key, quantity }),
  remove: (key: CartLineKey) => dispatchCart({ type: "remove", ...key }),
  clear: () => dispatchCart({ type: "clear" }),
};
