import type { ReactNode } from "react";
import { ContactForm } from "./contact-form";
import { site } from "@/shared/config";
import { NBSP } from "@/shared/lib";
import { AppLink, Button, PinIcon, cardClass } from "@/shared/ui";

const { phones, address, company, email, social } = site;

/** Opening hours stay on one line. */
const hours = <span className="whitespace-nowrap">10:00–23:00</span>;

/** "8 918 311-97-91" -> ["8 918", "311-97-91"] (the number is set on two lines in the big cards). */
const splitPhone = (display: string): [string, string] => {
  const [a, b, c] = display.split(NBSP);
  return [`${a}${NBSP}${b}`, c];
};

type PhoneCardProps = {
  href: string;
  label: { mobile: ReactNode; desktop: ReactNode };
  number: string;
  /** "или 8 (86155) 3-30-03" under the number (desktop). */
  alt?: string;
  variant?: "surface" | "accent";
};

/** A row with an arrow on mobile, a tall card with the number on two lines on desktop. */
function PhoneCard({ href, label, number, alt, variant = "surface" }: PhoneCardProps) {
  const [top, bottom] = splitPhone(number);
  const muted = variant === "surface" ? "text-text-muted" : "";

  return (
    <AppLink
      href={href}
      className={cardClass(
        variant,
        "grid grid-cols-[1fr_auto] items-center gap-x-4 rounded-md px-[18px] py-4 md:flex md:min-h-[234px] md:flex-col md:items-stretch md:justify-between md:gap-4 md:rounded-xl md:p-7",
      )}
    >
      <span className={`text-[13px] md:text-sm ${muted}`}>
        <span className="md:hidden">{label.mobile}</span>
        <span className="hidden md:inline">{label.desktop}</span>
      </span>
      <span className="col-start-1 font-display text-lg font-medium md:text-2xl md:leading-[1.15]">
        <span className="md:hidden">{number}</span>
        <span className="hidden md:inline">
          {top}
          <br />
          {bottom}
        </span>
      </span>
      {alt && <span className={`hidden text-sm md:block ${muted}`}>или {alt}</span>}
      <span aria-hidden="true" className="col-start-2 row-span-2 row-start-1 md:hidden">
        ↗
      </span>
    </AppLink>
  );
}

export function ContactsPage() {
  return (
    <main>
      <section className="mx-auto w-full max-w-page px-3 pt-4 md:px-6 md:pt-8">
        <h1 className="heading mx-1.5 my-2 text-4xl md:mx-0 md:mt-0 md:mb-6 md:text-[length:var(--text-h1)]">
          Контакты
        </h1>

        <div className="grid gap-2 md:grid-cols-4 md:gap-3">
          {/* Map placeholder, to be replaced by the Yandex Map */}
          <div className="relative flex h-[260px] items-center justify-center rounded-lg border border-placeholder-border bg-placeholder md:col-span-2 md:row-span-2 md:h-auto md:min-h-[480px] md:rounded-xl">
            <div className="flex flex-col items-center gap-2 text-text-muted">
              <span className="inline-flex size-[52px] items-center justify-center rounded-full bg-accent text-on-accent md:size-14">
                <PinIcon width={24} height={24} />
              </span>
              <strong className="hidden text-text md:block">Яндекс Карта</strong>
            </div>
            <div className="absolute inset-x-2.5 bottom-2.5 flex items-center justify-between gap-3 rounded-sm bg-surface px-3.5 py-3 md:inset-x-4 md:bottom-4 md:flex-wrap md:rounded-md md:px-5 md:py-4">
              <span className="flex flex-col">
                <strong>{address.street}</strong>
                <span className="text-[13px] text-text-muted md:text-sm">
                  <span className="md:hidden">у Городского парка</span>
                  <span className="hidden md:inline">
                    {address.postcode}, {address.city} · у Городского парка
                  </span>
                </span>
              </span>
              <Button href={site.mapUrl} variant="inverse" className="px-4 md:px-5">
                Маршрут<span className="hidden md:inline"> ↗</span>
              </Button>
            </div>
          </div>

          <PhoneCard
            href={phones.restaurant.href}
            label={{ mobile: <>Ресторан · {hours}</>, desktop: <>Ресторан и доставка · {hours}</> }}
            number={phones.restaurant.display}
            alt={phones.restaurant.alt.display}
          />
          <PhoneCard
            href={phones.hotel.href}
            label={{ mobile: "Отель · 24/7", desktop: "Отель · круглосуточно" }}
            number={phones.hotel.display}
            alt={phones.hotel.alt.display}
          />
          <PhoneCard
            href={phones.free.href}
            label={{ mobile: phones.free.label, desktop: phones.free.label }}
            number={phones.free.display}
            variant="accent"
          />

          {/* Online: a list on mobile, a card on desktop */}
          <div className={cardClass("inverse", "rounded-md px-[18px] py-1.5 md:hidden")}>
            {[
              { name: "Почта", value: email, href: `mailto:${email}` },
              ...social.map((s) => ({ name: s.name, value: s.handle, href: s.href })),
            ].map((row) => (
              <AppLink
                key={row.name}
                href={row.href}
                className="flex justify-between gap-3 border-b border-inverse-border py-3 last:border-b-0"
              >
                <span className="text-inverse-muted">{row.name}</span>
                <strong>{row.value}</strong>
              </AppLink>
            ))}
          </div>
          <div className={cardClass("inverse", "hidden min-h-[234px] justify-between gap-2.5 rounded-xl p-7 md:flex")}>
            <span className="text-sm text-inverse-muted">Онлайн</span>
            <AppLink href={`mailto:${email}`} className="tap-area font-semibold underline">
              {email}
            </AppLink>
            {social.map((s) => (
              <AppLink key={s.label} href={s.href} className="tap-area underline">
                {s.display}
              </AppLink>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto w-full max-w-page px-3 pt-10 md:grid md:grid-cols-3 md:items-start md:gap-3 md:px-6 md:pt-[104px]">
        <ContactForm className="md:col-span-2" />
        <div className={cardClass("surface", "mt-2 gap-2.5 rounded-lg p-6 md:mt-0 md:rounded-xl md:p-7")}>
          <h2 className="text-xl font-bold">Реквизиты</h2>
          <p className="text-[15px] text-text-muted">
            {company.name}
            <br />
            ИНН {company.inn}
            <br />
            ОГРН {company.ogrn}
            <br />
            {address.full}
            <br />
            тел. {phones.landline.display}
          </p>
          <p className="mt-1.5 text-[15px] text-text-muted">Для гостей — бесплатная парковка у здания.</p>
        </div>
      </section>
    </main>
  );
}
