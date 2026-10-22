import Image from "next/image";
import type { Room } from "@/entities/room";
import { StayDatesBar } from "@/features/search-rooms";
import { formatPrice, typo } from "@/shared/lib";
import { AppLink, Button, Card, Section } from "@/shared/ui";
import { FreeRooms } from "./free-rooms";

// The home page shows three short cards, with their own wording (not the same as the rooms page).
type Teaser = {
  roomId: string;
  name: string;
  details: { desktop: string; mobile: string };
  /** "от 4 000 ₽" */
  from?: boolean;
  photo: { src: string; alt: string };
  featured?: boolean;
};

const teasers: Teaser[] = [
  {
    roomId: "single",
    name: "Одноместный",
    details: { desktop: typo("19 м² · № 201 — с балконом"), mobile: typo("19 м²") },
    from: true,
    photo: { src: "/images/room-single.jpg", alt: "Одноместный номер" },
  },
  {
    roomId: "double-twin",
    name: "Двухместный",
    details: { desktop: typo("23–25 м² · 2 кровати или «евро»"), mobile: typo("23–25 м²") },
    photo: { src: "/images/room-double.jpg", alt: "Двухместный номер" },
  },
  {
    roomId: "vip",
    name: "VIP-номер",
    details: { desktop: typo("45 м² · 2 комнаты · балкон · до 4 гостей"), mobile: typo("45 м² · до 4 гостей") },
    photo: { src: "/images/room-vip.jpg", alt: "VIP-номер" },
    featured: true,
  },
];

export function RoomsSection({ rooms, free }: { rooms: readonly Room[]; free: number | null }) {
  const price = (roomId: string) => rooms.find((room) => room.id === roomId)?.pricePerNight ?? 0;

  return (
    <Section
      title="Номера"
      action={
        <>
          <AppLink href="/rooms" className="tap-area text-[15px] font-semibold md:hidden">
            Все ↗
          </AppLink>
          <StayDatesBar className="hidden md:flex" />
        </>
      }
    >
      {/* Mobile: horizontal rail that bleeds to the screen edge. From 768 px: grid. */}
      <div className="-mx-3 flex gap-2 overflow-x-auto px-3 pb-1 md:mx-0 md:grid md:grid-cols-2 md:gap-3 md:overflow-visible md:px-0 md:pb-0 lg:grid-cols-3">
        {teasers.map((teaser) => (
          <Card
            as="article"
            key={teaser.roomId}
            variant={teaser.featured ? "inverse" : "surface"}
            className="w-[270px] shrink-0 rounded-lg p-2 md:w-auto md:rounded-xl md:p-2.5"
          >
            {/* Mobile cards have no button: a link over the whole card instead. */}
            <AppLink
              href="/rooms"
              aria-label={`${teaser.name}: выбрать номер`}
              className="absolute inset-0 z-10 md:hidden"
            />
            <div className="relative h-[170px] overflow-hidden rounded-sm md:h-[250px] md:rounded-md">
              <Image
                src={teaser.photo.src}
                alt={teaser.photo.alt}
                fill
                sizes="(min-width: 1024px) 400px, (min-width: 768px) 50vw, 270px"
                className="object-cover"
              />
            </div>
            <div className="flex flex-1 flex-col gap-1.5 px-2 pt-3 pb-1.5 md:px-3.5 md:pt-[18px] md:pb-3">
              <h3 className="text-lg font-bold md:text-[22px]">{teaser.name}</h3>
              <p
                className={`flex-1 text-sm md:text-base ${teaser.featured ? "text-inverse-muted" : "text-text-muted"}`}
              >
                <span className="md:hidden">{teaser.details.mobile}</span>
                <span className="hidden md:inline">{teaser.details.desktop}</span>
              </p>
              <div className="mt-0.5 flex items-center justify-between gap-3 md:mt-3.5">
                <span className="font-display text-lg font-medium md:text-[22px] md:tracking-[-0.02em]">
                  {teaser.from && "от "}
                  {formatPrice(price(teaser.roomId))}
                </span>
                <Button
                  href="/rooms"
                  variant={teaser.featured ? "accent" : "inverse"}
                  className="hidden px-5 md:inline-flex"
                >
                  Выбрать
                </Button>
              </div>
            </div>
          </Card>
        ))}
      </div>
      {free !== null && <FreeRooms count={free} />}
    </Section>
  );
}
