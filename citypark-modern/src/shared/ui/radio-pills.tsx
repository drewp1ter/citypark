import type { ReactNode } from "react";
import { cn } from "@/shared/lib";

type Option = { value: string; label: ReactNode };

type RadioPillsProps = {
  name: string;
  /** Accessible name of the group (not shown). */
  legend: string;
  options: readonly Option[];
  value: string;
  onValueChange: (value: string) => void;
  /** "segmented": one rounded track with the selected option inside (Доставка / Самовывоз).
   *  "pills": separate outlined pills (payment method, topic). */
  variant?: "segmented" | "pills";
  className?: string;
};

/**
 * A group of radio buttons drawn as pills. The real <input type="radio"> is visually hidden but keeps
 * keyboard and screen reader behaviour; the selected pill gets the inverse colours.
 */
export function RadioPills({ name, legend, options, value, onValueChange, variant = "pills", className }: RadioPillsProps) {
  const segmented = variant === "segmented";

  return (
    <fieldset
      className={cn(
        "min-w-0",
        segmented ? "grid auto-cols-fr grid-flow-col rounded-pill bg-bg p-1 md:inline-grid md:self-start" : "flex flex-wrap gap-1.5 md:gap-2",
        className,
      )}
    >
      <legend className="sr-only">{legend}</legend>
      {options.map((option) => (
        <label
          key={option.value}
          className={cn(
            "inline-flex min-h-11 cursor-pointer items-center justify-center rounded-pill px-4 text-sm has-checked:bg-inverse-bg has-checked:font-semibold has-checked:text-inverse-text has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-accent md:text-base",
            segmented ? "md:px-[22px]" : "border border-border-strong has-checked:border-transparent",
          )}
        >
          <input
            type="radio"
            name={name}
            value={option.value}
            checked={value === option.value}
            onChange={() => onValueChange(option.value)}
            className="sr-only"
          />
          {option.label}
        </label>
      ))}
    </fieldset>
  );
}
