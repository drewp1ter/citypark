import { formatPrice } from "@/shared/lib";
import { delivery } from "../model/delivery";

export type DeliveryMethod = "delivery" | "pickup";
/** "city" = Белореченск, "outskirts" = СОТ and ДНТ. */
export type DeliveryZone = "city" | "outskirts";

export type DeliveryQuote = {
  /** Rubles. `null` = the price is not fixed, the operator confirms it by phone. */
  amount: number | null;
  /** What to show: "0 ₽", "200–350 ₽", "уточним". */
  label: string;
};

/**
 * Delivery price for an order, from the rules in the design:
 * pickup is free; in Белореченск delivery is free from 500 ₽; СОТ and ДНТ cost 200–350 ₽.
 *
 * Not in the design, so the defaults here are assumptions: an order below 500 ₽ in Белореченск has a price
 * the operator confirms ("уточним"), and the СОТ/ДНТ price is a range, so it is not added to the total.
 * TODO: replace the "уточним" cases with real tariffs when the owner sends them.
 */
export function quoteDelivery(input: { method: DeliveryMethod; zone: DeliveryZone; subtotal: number }): DeliveryQuote {
  if (input.method === "pickup") return { amount: 0, label: formatPrice(0) };
  if (input.zone === "outskirts") return { amount: null, label: delivery.outskirtsRange };
  if (input.subtotal >= delivery.freeFrom) return { amount: 0, label: formatPrice(0) };
  return { amount: null, label: "уточним" };
}
