import type { ReactNode } from "react";
import { site } from "@/shared/config";
import { delivery } from "@/entities/dish";

/** Checkout frame: no site navigation, just a quiet footer (the page brings its own header). */
export function CheckoutLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <div id="content" tabIndex={-1} className="flex-1 outline-none">
        {children}
      </div>
      <footer className="mx-auto hidden w-full max-w-narrow-page flex-wrap justify-between gap-3 px-6 pb-8 text-sm text-text-muted md:flex">
        <span>
          © {site.year} {site.company.name} · {site.address.street}
        </span>
        <span>Доставка {delivery.hours}</span>
      </footer>
    </div>
  );
}
