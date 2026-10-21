import type { ReactNode } from "react";
import { cn } from "@/shared/lib";
import { ChevronDownIcon } from "./icons";

type AccordionItemProps = {
  title: string;
  /** Open when the page loads. */
  defaultOpen?: boolean;
  /** Border and spacing of the item (differs between the desktop card and the mobile list). */
  className?: string;
  children: ReactNode;
};

/** One collapsible row. A native <details>, so it works without JavaScript and with the keyboard. */
export function AccordionItem({ title, defaultOpen = false, className, children }: AccordionItemProps) {
  return (
    <details open={defaultOpen} className={cn("group", className)}>
      <summary className="flex min-h-[52px] cursor-pointer list-none items-center justify-between gap-3 font-semibold [&::-webkit-details-marker]:hidden">
        {title}
        <ChevronDownIcon className="shrink-0 text-text-muted transition-transform group-open:rotate-180" />
      </summary>
      <div className="pb-4 text-[15px] text-text-muted">{children}</div>
    </details>
  );
}
