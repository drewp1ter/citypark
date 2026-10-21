"use client";

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { MenuCategory } from "@/entities/dish";
import { Chip, ChevronDownIcon, IconButton } from "@/shared/ui";

type CategoryTabsProps = {
  categories: readonly MenuCategory[];
  /** Id of the current category. */
  selectedId: string;
};

/** Pixels of tolerance when comparing scroll positions (they can be fractional). */
const EDGE = 1;

/**
 * Category chips as a sticky row. Swiped below 768 px; from 768 px «‹» / «›» scroll it and the chips fade out
 * at an edge with more to scroll. The chips are links (/menu?category=…), so the server knows the category.
 */
export function CategoryTabs({ categories, selectedId }: CategoryTabsProps) {
  const scroller = useRef<HTMLDivElement>(null);
  const [edges, setEdges] = useState({ left: false, right: false });

  const measure = useCallback(() => {
    const el = scroller.current;
    if (!el) return;
    const left = el.scrollLeft > EDGE;
    const right = el.scrollLeft + el.clientWidth < el.scrollWidth - EDGE;
    setEdges((prev) => (prev.left === left && prev.right === right ? prev : { left, right }));
  }, []);

  // Bring the selected chip into view before the first paint. scrollIntoView could move the page too.
  useLayoutEffect(() => {
    const el = scroller.current;
    const chip = el?.querySelector<HTMLElement>("[aria-current]");
    if (!el || !chip) return;
    const chipLeft = chip.getBoundingClientRect().left - el.getBoundingClientRect().left + el.scrollLeft;
    el.scrollLeft = chipLeft - (el.clientWidth - chip.offsetWidth) / 2;
    measure();
  }, [measure]);

  useEffect(() => {
    const el = scroller.current;
    if (!el) return;
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    return () => observer.disconnect();
  }, [measure]);

  function scrollBy(direction: -1 | 1) {
    const el = scroller.current;
    if (!el) return;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    el.scrollBy({ left: direction * el.clientWidth * 0.8, behavior: reduce ? "auto" : "smooth" });
  }

  const arrow = "absolute top-1/2 hidden -translate-y-1/2 md:inline-flex";

  return (
    <nav
      aria-label="Разделы меню"
      className="sticky top-[74px] z-[5] -mx-3 min-w-0 bg-bg px-3 py-2 md:top-[84px] md:mr-0 md:-ml-2 md:pr-0 md:pl-2"
    >
      <div className="relative">
        <div
          ref={scroller}
          onScroll={measure}
          style={{ "--fade-l": edges.left ? "48px" : "0px", "--fade-r": edges.right ? "48px" : "0px" } as React.CSSProperties}
          className="no-scrollbar flex gap-1.5 overflow-x-auto md:scroll-px-14 md:[mask-image:linear-gradient(to_right,transparent,black_var(--fade-l),black_calc(100%_-_var(--fade-r)),transparent)]"
        >
          {categories.map((item, index) => (
            <Chip
              key={item.id}
              href={index === 0 ? "/menu" : `/menu?category=${item.id}`}
              scroll={false}
              selected={item.id === selectedId}
              className="px-4"
            >
              {item.name}
            </Chip>
          ))}
        </div>
        {edges.left && (
          <IconButton
            variant="surface"
            aria-label="Прокрутить разделы влево"
            aria-hidden="true"
            tabIndex={-1}
            onClick={() => scrollBy(-1)}
            className={`${arrow} left-0`}
          >
            <ChevronDownIcon className="rotate-90" />
          </IconButton>
        )}
        {edges.right && (
          <IconButton
            variant="surface"
            aria-label="Прокрутить разделы вправо"
            aria-hidden="true"
            tabIndex={-1}
            onClick={() => scrollBy(1)}
            className={`${arrow} right-0`}
          >
            <ChevronDownIcon className="-rotate-90" />
          </IconButton>
        )}
      </div>
    </nav>
  );
}
