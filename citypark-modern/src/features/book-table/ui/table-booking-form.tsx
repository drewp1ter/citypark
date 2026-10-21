"use client";

import { useActionState } from "react";
import { cn, useToday } from "@/shared/lib";
import { Button, PhoneField, SelectField, TextField, cardClass } from "@/shared/ui";
import { submitBooking } from "../api/submit-booking";
import { guestOptions, halls, type BookingState } from "../model/schema";

const initialState: BookingState = { status: "idle" };

type TableBookingFormProps = {
  /**
   * "card": a white card with its own heading (home page).
   * "dark": just the fields, for the dark booking block of the restaurant page; adds the "Зал" field.
   */
  variant?: "card" | "dark";
  className?: string;
};

/** "Забронировать столик": date, time, guests, phone. Validated on the server, then sent through `submitLead`. */
export function TableBookingForm({ variant = "card", className }: TableBookingFormProps) {
  const [state, formAction, pending] = useActionState(submitBooking, initialState);
  const today = useToday();
  const dark = variant === "dark";
  const tone = dark ? "inverse" : "default";

  const errors = state.status === "error" ? state.errors : {};
  const values = state.status === "error" ? state.values : undefined;

  return (
    <form
      action={formAction}
      noValidate
      className={
        dark
          ? cn("grid grid-cols-2 content-start gap-2.5 md:gap-3", className)
          : cardClass("surface", `gap-3.5 rounded-xl p-7 ${className ?? ""}`)
      }
    >
      {dark ? (
        // On desktop the heading sits in the left column of the block; on mobile it is the first line of the form.
        <h2 className="col-span-2 mb-1 font-display text-2xl font-medium tracking-[-0.02em] md:hidden">
          Забронировать столик
        </h2>
      ) : (
        <h3 className="text-2xl font-bold">Забронировать столик</h3>
      )}

      {state.status === "success" ? (
        <p role="status" className={cn("col-span-2 text-[15px]", dark ? "text-inverse-muted" : "text-text-muted")}>
          Заявка отправлена. Перезвоним вам в ближайшее время, чтобы подтвердить столик.
        </p>
      ) : (
        <>
          {!dark && <p className="text-[15px] text-text-muted">Ресторан или летнее кафе. Перезвоним.</p>}
          <div
            className={cn(
              "grid gap-3.5 md:grid-cols-2 lg:grid-cols-1",
              dark && "col-span-2 grid-cols-2 gap-2.5 md:gap-3 lg:grid-cols-2",
            )}
          >
            <TextField
              tone={tone}
              label="Дата"
              type="date"
              name="date"
              min={today || undefined}
              defaultValue={values?.date ?? today}
              error={errors.date}
            />
            <TextField
              tone={tone}
              label="Время"
              type="time"
              name="time"
              defaultValue={values?.time ?? "19:00"}
              error={errors.time}
            />
            <SelectField
              tone={tone}
              label="Гостей"
              name="guests"
              defaultValue={values?.guests ?? guestOptions[0]}
              error={errors.guests}
            >
              {guestOptions.map((option) => (
                <option key={option}>{option}</option>
              ))}
            </SelectField>
            {dark && (
              <SelectField
                tone={tone}
                label="Зал"
                name="hall"
                defaultValue={values?.hall || halls[0].value}
                error={errors.hall}
              >
                {halls.map((hall) => (
                  <option key={hall.value} value={hall.value}>
                    {hall.label}
                  </option>
                ))}
              </SelectField>
            )}
            <PhoneField
              tone={tone}
              label="Телефон"
              name="phone"
              defaultValue={values?.phone}
              error={errors.phone}
              className={dark ? "col-span-2" : undefined}
            />
          </div>
          <Button
            type="submit"
            variant={dark ? "accent" : "inverse"}
            size="xl"
            disabled={pending}
            className={cn("w-full", dark ? "col-span-2 mt-1 md:mt-0 md:min-h-[54px]" : "mt-auto")}
          >
            {pending ? (
              "Отправляем…"
            ) : dark ? (
              <>
                <span className="md:hidden">Отправить</span>
                <span className="max-md:hidden">Отправить заявку</span>
              </>
            ) : (
              "Отправить заявку"
            )}
          </Button>
        </>
      )}
    </form>
  );
}
