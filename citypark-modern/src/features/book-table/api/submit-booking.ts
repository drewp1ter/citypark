"use server";

import { submitLead } from "@/shared/api";
import { phoneDigits, todayInMoscow } from "@/shared/lib";
import { bookingSchema, halls, type BookingField, type BookingState } from "../model/schema";

const fields: BookingField[] = ["date", "time", "guests", "phone", "hall"];

export async function submitBooking(_previous: BookingState, formData: FormData): Promise<BookingState> {
  const values = Object.fromEntries(fields.map((field) => [field, String(formData.get(field) ?? "")])) as Record<
    BookingField,
    string
  >;

  const errors: Partial<Record<BookingField, string>> = {};
  const parsed = bookingSchema.safeParse(values);
  if (!parsed.success) {
    for (const issue of parsed.error.issues) {
      const field = issue.path[0] as BookingField;
      errors[field] ??= issue.message;
    }
  }
  if (!errors.date && values.date < todayInMoscow()) errors.date = "Дата не может быть в прошлом";
  if (!parsed.success || Object.keys(errors).length > 0) return { status: "error", errors, values };

  const { hall, ...booking } = parsed.data;
  await submitLead("table-booking", {
    ...booking,
    hall: halls.find((item) => item.value === hall)?.label ?? null,
    phone: `+${phoneDigits(booking.phone)}`,
  });
  return { status: "success" };
}
