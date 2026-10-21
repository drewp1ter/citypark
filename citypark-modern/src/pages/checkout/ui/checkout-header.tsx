import { site } from "@/shared/config";
import { AppLink, IconButton } from "@/shared/ui";

/**
 * Header of the checkout steps. It has no site navigation, to keep the focus on the order:
 * a back link, the logo and the phone. On mobile it is a sticky bar with the step counter.
 */
export function CheckoutHeader({ step }: { step: 2 | 3 }) {
  const phone = site.phones.restaurant;

  return (
    <header className="sticky top-0 z-10 bg-bg px-3 pt-2.5 pb-1.5 md:static md:bg-transparent md:px-6 md:pt-4 md:pb-0">
      <div className="flex items-center gap-2.5 md:hidden">
        <IconButton variant="surface" href="/menu" aria-label="Назад в меню">
          <span aria-hidden="true" className="text-lg">
            ←
          </span>
        </IconButton>
        <p aria-hidden="true" className="flex-1 font-display text-xl font-medium">
          Оформление
        </p>
        <span className="rounded-pill bg-inverse-bg px-2.5 py-1 text-xs font-semibold text-inverse-text">
          {step} / 3
        </span>
      </div>

      <div className="mx-auto hidden max-w-[calc(var(--container-narrow)+22px)] items-center justify-between gap-3 rounded-pill border border-border bg-surface py-2 pr-2 pl-3 md:flex">
        <AppLink href="/menu" className="inline-flex min-h-11 items-center gap-2 rounded-pill bg-bg px-3 font-semibold">
          <span aria-hidden="true">←</span> В меню
        </AppLink>
        <AppLink href="/" className="inline-flex min-h-11 items-center font-display text-lg font-semibold">
          {site.wordmark}
        </AppLink>
        <AppLink href={phone.href} className="inline-flex min-h-11 items-center px-3 font-semibold">
          {phone.display}
        </AppLink>
      </div>
    </header>
  );
}
