import { typo } from "@/shared/lib";

/** Delivery facts shown on the home page, the menu page, the cart and checkout. */
export const delivery = {
  /** Order total (rubles) from which delivery in Белореченск is free. */
  freeFrom: 500,
  hours: "10:00–23:00",
  cooking: typo("~30 мин"),
  outskirtsFee: typo("СОТ и ДНТ 200–350 ₽"),
  /** Just the range, for the order summary. */
  outskirtsRange: typo("200–350 ₽"),
} as const;
