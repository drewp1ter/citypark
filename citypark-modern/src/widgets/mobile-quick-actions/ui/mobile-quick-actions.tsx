"use client";

import { usePathname } from "next/navigation";
import { site } from "@/shared/config";
import { Button } from "@/shared/ui";

/**
 * Sticky bar at the bottom of mobile pages. On /rooms it calls the hotel and the free number, on /hotel it calls
 * the hotel and leads to the rooms, on /restaurant it jumps to the booking form, on /news it opens the menu and the
 * rooms; elsewhere it calls the restaurant and leads to the rooms.
 */
export function MobileQuickActions() {
  const pathname = usePathname() ?? "";
  const rooms = pathname.startsWith("/rooms");
  const hotel = rooms || pathname.startsWith("/hotel");
  const restaurant = pathname.startsWith("/restaurant");
  const news = pathname.startsWith("/news");

  return (
    <nav
      aria-label="Быстрые действия"
      className="sticky bottom-0 z-10 grid grid-cols-2 gap-2 bg-linear-to-b from-transparent to-bg to-35% px-3 pt-2.5 pb-3 md:hidden"
    >
      {news ? (
        <Button href="/menu" variant="surface" size="xl">
          Меню
        </Button>
      ) : (
        <Button href={(hotel ? site.phones.hotel : site.phones.restaurant).href} variant="surface" size="xl">
          Позвонить
        </Button>
      )}
      {rooms ? (
        <Button href={site.phones.free.href} variant="inverse" size="xl" className="px-3 whitespace-nowrap">
          8 800 · бесплатно
        </Button>
      ) : restaurant ? (
        <Button href="#booking" variant="inverse" size="xl">
          Столик
        </Button>
      ) : (
        <Button href="/rooms" variant="inverse" size="xl">
          {hotel ? "Выбрать номер" : "Забронировать"}
        </Button>
      )}
    </nav>
  );
}
