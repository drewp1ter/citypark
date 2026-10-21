import { AppLink } from "./app-link";

type Crumb = { label: string; href?: string };

/** "Главная / Отель / Номера". The last item is the current page and has no link. */
export function Breadcrumbs({ items }: { items: readonly Crumb[] }) {
  return (
    <nav aria-label="Хлебные крошки" className="mb-5 text-sm text-text-muted">
      <ol className="flex flex-wrap gap-x-1.5">
        {items.map((item, index) => (
          <li key={item.label} className="flex gap-1.5" aria-current={item.href ? undefined : "page"}>
            {item.href ? (
              <AppLink href={item.href} className="tap-area underline">
                {item.label}
              </AppLink>
            ) : (
              item.label
            )}
            {index < items.length - 1 && <span aria-hidden="true">/</span>}
          </li>
        ))}
      </ol>
    </nav>
  );
}
