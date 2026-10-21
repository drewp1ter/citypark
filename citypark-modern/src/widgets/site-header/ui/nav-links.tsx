"use client";

import { usePathname } from "next/navigation";
import { site } from "@/shared/config";
import { cn } from "@/shared/lib";
import { AppLink } from "@/shared/ui";

/** Desktop navigation. The link of the current section gets the "pill" state from the design. */
export function NavLinks() {
  const pathname = usePathname() ?? "";

  return (
    <nav aria-label="Основное меню" className="flex gap-x-1.5 text-[15px]">
      {site.nav.map(({ href, label }) => {
        const active = pathname === href || pathname.startsWith(`${href}/`);
        return (
          <AppLink
            key={href}
            href={href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "inline-flex min-h-11 items-center rounded-pill px-3.5",
              active ? "bg-bg font-semibold" : "font-medium",
            )}
          >
            {label}
          </AppLink>
        );
      })}
    </nav>
  );
}
