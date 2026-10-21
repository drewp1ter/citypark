"use client";

import { useState, type ComponentProps } from "react";
import { formatPhoneInput } from "@/shared/lib";
import { TextField } from "./fields";

type PhoneFieldProps = Omit<ComponentProps<typeof TextField>, "type" | "value" | "onChange" | "defaultValue"> & {
  defaultValue?: string;
};

/** Phone input with the "+7 ___ ___-__-__" mask. The submitted value is the formatted text. */
export function PhoneField({ defaultValue = "", ...props }: PhoneFieldProps) {
  const [value, setValue] = useState(formatPhoneInput(defaultValue));

  return (
    <TextField
      {...props}
      type="tel"
      inputMode="tel"
      autoComplete="tel"
      placeholder="+7 ___ ___-__-__"
      value={value}
      onChange={(event) => setValue(formatPhoneInput(event.target.value))}
    />
  );
}
