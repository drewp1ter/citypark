import { z } from "zod";
import { isValidRuPhone } from "@/shared/lib";

export const guestOptions = ["2", "3–4", "5–8", "Банкет"] as const;

/** Where to sit. Only the restaurant page asks; the home form leaves it empty. */
export const halls = [
  { value: "restaurant", label: "Ресторан" },
  { value: "cafe", label: "Летнее кафе" },
] as const;

export const bookingSchema = z.object({
  date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Выберите дату"),
  time: z.string().regex(/^\d{2}:\d{2}$/, "Выберите время"),
  guests: z.enum(guestOptions, { error: "Выберите количество гостей" }),
  phone: z.string().refine(isValidRuPhone, "Введите номер полностью: +7 ___ ___-__-__"),
  hall: z.enum(["restaurant", "cafe", ""], { error: "Выберите зал" }),
});

export type BookingField = keyof z.infer<typeof bookingSchema>;

/** What the form shows after each submit. On an error the typed values come back, so nothing is lost. */
export type BookingState =
  | { status: "idle" }
  | { status: "success" }
  | { status: "error"; errors: Partial<Record<BookingField, string>>; values: Record<BookingField, string> };
