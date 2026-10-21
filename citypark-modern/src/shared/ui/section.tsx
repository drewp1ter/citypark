import { useId, type ReactNode } from "react";
import { cn } from "@/shared/lib";

type SectionProps = {
  id?: string;
  /** The h2. Leave it out for a section without a title (e.g. the contacts block). */
  title?: string;
  /** Right side of the title row: a link ("Все новости ↗"), chips or a form. */
  action?: ReactNode;
  className?: string;
  children: ReactNode;
};

/** Page section: max width 1240 px, side gutters, big top gap, optional title row. */
export function Section({ id, title, action, className, children }: SectionProps) {
  const headingId = useId();

  return (
    <section
      id={id}
      aria-labelledby={title ? headingId : undefined}
      className={cn("mx-auto w-full max-w-page px-3 pt-14 md:px-6 md:pt-28", className)}
    >
      {title && (
        <div className="mb-3.5 flex flex-wrap items-end justify-between gap-4 md:mb-7">
          <h2 id={headingId} className="heading text-[32px] md:text-[40px] lg:text-[length:var(--text-h2)]">
            {title}
          </h2>
          {action}
        </div>
      )}
      {children}
    </section>
  );
}
