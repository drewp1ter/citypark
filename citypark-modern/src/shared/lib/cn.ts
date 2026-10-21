import { twMerge } from "tailwind-merge";

/** Joins class names, skips falsy values, and lets a later Tailwind class win over an earlier one that sets the same property. */
export function cn(...values: Array<string | false | null | undefined>): string {
  return twMerge(values.filter(Boolean).join(" "));
}
