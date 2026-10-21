"use client";

import type { ReactNode } from "react";
import {
  Button,
  Label,
  ListBox,
  ListBoxItem,
  Popover,
  Select as AriaSelect,
  SelectValue,
  Text,
} from "react-aria-components";
import { cn } from "@/shared/lib";
import { controlClass, fieldErrorClass, fieldLabelClass, inverseControlClass, type FieldTone } from "./fields";
import { ChevronDownIcon } from "./icons";

export type SelectOption = {
  value: string;
  label: ReactNode;
  /** Plain text for type-ahead when the label is markup. */
  textValue?: string;
};

type SelectProps = {
  label: ReactNode;
  name: string;
  options: readonly SelectOption[];
  /** Controlled value. */
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  error?: string;
  tone?: FieldTone;
  className?: string;
  required?: boolean;
  disabled?: boolean;
};

/**
 * Dropdown list with our own look. Behaviour (keyboard, ARIA, focus, positioning) comes from React Aria; with `name`
 * it also renders a hidden native field, so forms post the same values as a native select element.
 */
export function Select({
  label,
  name,
  options,
  value,
  defaultValue,
  onValueChange,
  error,
  tone = "default",
  className,
  required,
  disabled,
}: SelectProps) {
  const inverse = tone === "inverse";

  return (
    <AriaSelect
      name={name}
      selectedKey={value}
      defaultSelectedKey={defaultValue}
      onSelectionChange={(key) => onValueChange?.(String(key))}
      isInvalid={Boolean(error)}
      isRequired={required}
      isDisabled={disabled}
      className={cn(fieldLabelClass(tone), className)}
    >
      <Label>{label}</Label>
      <Button
        className={cn(
          inverse ? inverseControlClass : controlClass,
          "group flex items-center justify-between gap-2 text-left data-[focus-visible]:outline-2 data-[focus-visible]:outline-offset-2 data-[focus-visible]:outline-accent",
          inverse ? "data-[invalid]:border-danger-on-inverse" : "data-[invalid]:border-danger",
        )}
      >
        <SelectValue className="truncate" />
        <ChevronDownIcon aria-hidden className="shrink-0 motion-safe:transition-transform group-aria-expanded:rotate-180" />
      </Button>
      {error && (
        <Text slot="errorMessage" role="alert" className={fieldErrorClass(tone)}>
          {error}
        </Text>
      )}
      <Popover
        offset={6}
        className={cn(
          "w-(--trigger-width) overflow-auto rounded-xs border p-1 text-base font-normal max-h-72",
          inverse
            ? "border-inverse-border-strong bg-inverse-surface text-inverse-text"
            : "border-border bg-surface text-text",
          "data-[entering]:motion-safe:animate-[select-in_120ms_ease-out] data-[exiting]:motion-safe:animate-[select-out_90ms_ease-in]",
        )}
      >
        <ListBox className="outline-none">
          {options.map((option) => (
            <ListBoxItem
              key={option.value}
              id={option.value}
              textValue={option.textValue ?? (typeof option.label === "string" ? option.label : option.value)}
              className={cn(
                "flex min-h-11 cursor-pointer items-center rounded-[12px] px-3",
                inverse
                  ? "data-[focused]:bg-inverse-bg data-[selected]:bg-accent data-[selected]:font-semibold data-[selected]:text-on-accent"
                  : "data-[focused]:bg-bg data-[selected]:bg-inverse-bg data-[selected]:font-semibold data-[selected]:text-inverse-text",
                "data-[focus-visible]:outline-2 data-[focus-visible]:-outline-offset-2 data-[focus-visible]:outline-accent",
              )}
            >
              {option.label}
            </ListBoxItem>
          ))}
        </ListBox>
      </Popover>
    </AriaSelect>
  );
}
