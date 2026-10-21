"use client";

import { useRef, useState, type ComponentPropsWithoutRef } from "react";
import { cn } from "@/shared/lib";
import { ClearButton } from "./clear-button";
import type { FieldTone } from "./fields";

type ClearableInputProps = Omit<ComponentPropsWithoutRef<"input">, "className"> & {
  className: string;
  tone: FieldTone;
  onClear?: () => void;
};

/** Input with a «×» button while it has text. Works controlled (`value`) and uncontrolled (`defaultValue`). */
export function ClearableInput({ className, tone, onClear, onChange, ...props }: ClearableInputProps) {
  const ref = useRef<HTMLInputElement>(null);
  const [hasText, setHasText] = useState(() => String(props.value ?? props.defaultValue ?? "") !== "");
  const controlled = props.value !== undefined;
  const showButton = controlled ? String(props.value) !== "" : hasText;

  const clear = () => {
    const input = ref.current;
    if (input && !controlled) {
      input.value = "";
      input.dispatchEvent(new Event("input", { bubbles: true }));
      setHasText(false);
    }
    onClear?.();
    input?.focus();
  };

  return (
    <div className="relative">
      <input
        {...props}
        ref={ref}
        onChange={(event) => {
          setHasText(event.target.value !== "");
          onChange?.(event);
        }}
        className={cn(className, "pr-12")}
      />
      {showButton && <ClearButton tone={tone} onClick={clear} className="absolute right-0 top-1/2 -translate-y-1/2" />}
    </div>
  );
}
