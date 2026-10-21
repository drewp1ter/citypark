import Image from "next/image";
import type { NewsItem } from "@/entities/article";
import { formatDate } from "@/shared/lib";
import { AppLink, cardClass } from "@/shared/ui";
import { SectionTabs } from "./section-tabs";

const href = (item: NewsItem) => `/news/${item.slug}`;

function LatestCard({ item }: { item: NewsItem }) {
  return (
    <AppLink
      href={href(item)}
      className="relative block h-[320px] overflow-hidden rounded-lg md:col-span-2 md:min-h-[480px] md:rounded-xl lg:row-span-2"
    >
      <Image
        src={item.photo.src}
        alt={item.photo.alt}
        fill
        loading="eager"
        fetchPriority="high"
        sizes="(min-width: 1280px) 820px, 100vw"
        className="object-cover"
      />
      <span className="absolute inset-x-2.5 bottom-2.5 flex flex-col gap-1.5 rounded-sm bg-surface p-4 md:inset-x-4 md:bottom-4 md:gap-2 md:rounded-md md:p-6">
        <span className="flex items-center gap-2">
          <span className="rounded-pill bg-accent px-2.5 py-[3px] text-xs font-semibold text-on-accent md:py-1 md:text-[13px]">
            Последняя
          </span>
          <span className="text-[13px] text-text-muted md:text-sm">{formatDate(item.date)}</span>
        </span>
        <span className="font-display text-xl leading-[1.15] font-medium md:text-[32px] md:leading-[1.1] md:tracking-[-0.02em]">
          {item.title}
        </span>
        {item.excerpt && <span className="text-text-muted max-md:hidden">{item.excerpt}</span>}
      </span>
    </AppLink>
  );
}

/** Desktop: a photo card with date and title. */
function NewsCard({ item }: { item: NewsItem }) {
  return (
    <AppLink href={href(item)} className={cardClass("surface", "rounded-lg p-2.5 max-md:hidden")}>
      <div className="relative h-[140px] overflow-hidden rounded-sm">
        <Image
          src={item.photo.src}
          alt={item.photo.alt}
          fill
          sizes="(min-width: 1024px) 400px, 50vw"
          className="object-cover"
        />
      </div>
      <span className="flex flex-col gap-1 px-2.5 pt-3.5 pb-2">
        <span className="text-sm text-text-muted">{formatDate(item.date)}</span>
        <span className="text-[19px] font-bold">{item.title}</span>
      </span>
    </AppLink>
  );
}

/** The latest news as a big card, the next ones as cards (desktop) or rows with a thumbnail (mobile). */
export function NewsFeed({ items }: { items: readonly NewsItem[] }) {
  const [latest, ...rest] = items;
  const cards = rest.slice(0, 2);
  const rows = rest.slice(2);
  // The mobile list shows five items after the latest one; the oldest news is left out, as in the design.
  const mobileRows = rest.slice(0, 5);

  return (
    <section id="news" className="mx-auto w-full max-w-page scroll-mt-24 px-3 pt-4 md:px-6 md:pt-8">
      <div className="flex flex-wrap items-end justify-between gap-5 md:mb-6">
        <h1 className="heading mx-1.5 mt-2 mb-3.5 text-4xl max-md:w-full md:mx-0 md:mt-0 md:mb-0 md:text-[length:var(--text-h1)]">
          Новости
        </h1>
        <SectionTabs />
      </div>

      <div className="mt-3 grid gap-3 md:mt-0 md:grid-cols-2 lg:grid-cols-3">
        <LatestCard item={latest} />
        {cards.map((item) => (
          <NewsCard key={item.slug} item={item} />
        ))}
      </div>

      {/* Desktop: the older news as rows */}
      <ul className="mt-3 rounded-lg border border-border bg-surface px-7 py-2 max-md:hidden">
        {rows.map((item) => (
          <li key={item.slug} className="border-b border-divider last:border-b-0">
            <AppLink href={href(item)} className="grid grid-cols-[140px_1fr_auto] items-center gap-5 py-[18px]">
              <span className="text-text-muted">{formatDate(item.date)}</span>
              <span className="text-[19px] font-bold">{item.title}</span>
              <span aria-hidden="true">↗</span>
            </AppLink>
          </li>
        ))}
      </ul>

      {/* Mobile: every news item as a row with a thumbnail */}
      <ul className="mt-2 rounded-md border border-border bg-surface px-4 py-1 md:hidden">
        {mobileRows.map((item) => (
          <li key={item.slug} className="border-b border-divider last:border-b-0">
            <AppLink href={href(item)} className="flex items-center gap-3 py-3">
              <Image
                src={item.photo.src}
                alt=""
                width={64}
                height={64}
                className="size-16 shrink-0 rounded-2xs object-cover"
              />
              <span className="flex flex-col">
                <span className="text-[13px] text-text-muted">{formatDate(item.date)}</span>
                <strong>{item.shortTitle ?? item.title}</strong>
              </span>
            </AppLink>
          </li>
        ))}
      </ul>
    </section>
  );
}
