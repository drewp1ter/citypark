"use client";

import { useEffect } from "react";
import { cartActions } from "@/entities/cart";

/** Empties the cart once the order has been placed (this page is where the order action redirects to). */
export function ClearCart() {
  useEffect(() => {
    cartActions.clear();
  }, []);
  return null;
}
