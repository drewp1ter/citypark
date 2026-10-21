"use server";

import { redirect } from "next/navigation";
import { getMenu, priceLines, quoteDelivery } from "@/entities/dish";
import { submitLead } from "@/shared/api";
import { phoneDigits } from "@/shared/lib";
import {
  conditionalErrors,
  linesSchema,
  orderFields,
  orderSchema,
  type OrderErrorKey,
  type OrderField,
  type OrderState,
} from "../model/schema";

/**
 * Validates the order, prices it from the menu (never from the browser) and sends it through `submitLead`.
 * On success it redirects to the "Готово" step; the cart is cleared there.
 */
export async function placeOrder(_previous: OrderState, formData: FormData): Promise<OrderState> {
  const values = Object.fromEntries(orderFields.map((field) => [field, String(formData.get(field) ?? "")])) as Record<
    OrderField,
    string
  >;
  const errors: Partial<Record<OrderErrorKey, string>> = {};

  const order = orderSchema.safeParse(values);
  if (!order.success) {
    for (const issue of order.error.issues) {
      const field = issue.path[0] as OrderField;
      errors[field] ??= issue.message;
    }
  }
  for (const [field, message] of Object.entries(conditionalErrors(values)) as [OrderField, string][]) {
    errors[field] ??= message;
  }

  let rawLines: unknown = null;
  try {
    rawLines = JSON.parse(String(formData.get("lines") ?? "[]"));
  } catch {
    // handled below: the schema rejects null
  }
  const lines = linesSchema.safeParse(rawLines);
  if (!lines.success) errors.form = "Корзина пуста. Добавьте блюда из меню.";

  if (!order.success || !lines.success || Object.keys(errors).length > 0) return { status: "error", errors, values };

  let menu;
  try {
    menu = await getMenu();
  } catch (error) {
    console.error(error);
    errors.form = "Не удалось проверить меню. Попробуйте ещё раз через минуту.";
    return { status: "error", errors, values };
  }
  const priced = priceLines(lines.data, menu.dishes);
  if (priced.items.length !== lines.data.length) {
    // A dish was removed from the menu, is on the stop-list or has no price since it was put in the cart.
    errors.form = "Меню изменилось: некоторых блюд из корзины больше нет. Проверьте заказ.";
    return { status: "error", errors, values };
  }

  const quote = quoteDelivery({ method: order.data.method, zone: order.data.zone, subtotal: priced.subtotal });
  await submitLead("order", {
    ...order.data,
    consent: true,
    phone: `+${phoneDigits(order.data.phone)}`,
    items: priced.items.map((item) => ({
      dish: item.dish.name,
      size: item.variant.label,
      quantity: item.quantity,
      total: item.total,
    })),
    subtotal: priced.subtotal,
    delivery: quote.label,
    total: priced.subtotal + (quote.amount ?? 0),
  });

  // redirect() works by throwing, so it must stay outside any try/catch.
  redirect("/checkout/done");
}
