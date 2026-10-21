import Image from "next/image";
import type { Room, RoomAvailability } from "@/entities/room";
import { cn, formatPrice } from "@/shared/lib";
import { Button, Card } from "@/shared/ui";

// TODO: "Выбрать" scrolls to the search bar, as in the design. When a booking engine is connected it opens the booking.
const BOOK_HREF = "#search";

function statusLabel(availability: RoomAvailability | undefined): string {
  if (!availability || availability.state === "unavailable") return "Нет на эти даты";
  return availability.left ? `Свободно ${availability.left}` : "Свободен";
}

function StatusBadge({ available, children, className }: { available: boolean; children: string; className?: string }) {
  return (
    <span
      className={cn(
        "rounded-pill px-2.5 py-1 text-xs font-semibold md:px-3 md:text-[13px]",
        available ? "bg-accent text-on-accent" : "bg-surface text-text",
        className,
      )}
    >
      {children}
    </span>
  );
}

type RoomCardProps = {
  /** The first card of the page: its photo is the LCP element on mobile, so it loads eagerly. */
  first?: boolean;
  room: Room;
  availability: RoomAvailability | undefined;
  nights: number;
};

function RoomPhoto({
  room,
  available,
  eager,
  className,
}: {
  room: Room;
  available: boolean;
  eager?: boolean;
  className: string;
}) {
  return (
    <div className={cn("relative overflow-hidden", className)}>
      <Image
        src={room.photo.src}
        alt={room.photo.alt}
        fill
        loading={eager ? "eager" : undefined}
        fetchPriority={eager ? "high" : undefined}
        sizes="(min-width: 1024px) 400px, (min-width: 768px) 50vw, 100vw"
        style={room.photo.position ? { objectPosition: room.photo.position } : undefined}
        className={cn("object-cover", !available && "grayscale-50")}
      />
    </div>
  );
}

/** The VIP room: a dark card, two columns wide on desktop with a photo on the left and the details on the right. */
function FeaturedRoomCard({ room, availability, nights, first }: RoomCardProps) {
  const available = availability?.state === "available";
  const status = statusLabel(availability);

  return (
    <Card
      as="article"
      variant="inverse"
      className="rounded-lg p-2 md:col-span-2 md:flex-row md:flex-wrap md:gap-2.5 md:rounded-xl md:p-2.5"
    >
      <div className="relative md:min-w-0 md:flex-[1_1_380px]">
        <RoomPhoto
          room={room}
          available={available}
          eager={first}
          className="h-[210px] rounded-sm md:h-[360px] md:rounded-md"
        />
        <StatusBadge available={available} className="absolute top-2.5 left-2.5 md:hidden">
          {status}
        </StatusBadge>
      </div>
      <div className="flex min-w-0 flex-col gap-1 px-2.5 pt-3.5 pb-2 md:flex-[1_1_280px] md:gap-2.5 md:px-5 md:py-[22px]">
        <StatusBadge available={available} className="hidden self-start md:inline-block">
          {status}
        </StatusBadge>
        <h2 className="font-display text-[22px] font-medium md:text-[32px] md:tracking-[-0.02em]">{room.name}</h2>
        <p className="text-sm text-inverse-muted md:text-base">
          {room.numbers} · <span className="md:hidden">{room.area}</span>
          <span className="hidden md:inline">{room.areaDetailed ?? room.area}</span>
          {room.capacity && ` · ${room.capacity}`}
        </p>
        {room.description && <p className="hidden flex-1 md:block">{room.description}</p>}

        <div className="mt-2 flex items-center justify-between md:hidden">
          <span className="font-display text-xl font-medium">{formatPrice(room.pricePerNight)}</span>
          <Button href={BOOK_HREF} variant="accent">
            Выбрать
          </Button>
        </div>
        <p className="hidden md:block">
          <span className="font-display text-[30px] font-medium">{formatPrice(room.pricePerNight)}</span>
          <span className="text-inverse-muted"> / ночь</span>
        </p>
        <Button href={BOOK_HREF} variant="accent" className="hidden min-h-[50px] md:inline-flex">
          Забронировать · {formatPrice(room.pricePerNight * nights)}
        </Button>
      </div>
    </Card>
  );
}

export function RoomCard(props: RoomCardProps) {
  const { room, availability, first } = props;
  if (room.kind === "vip") return <FeaturedRoomCard {...props} />;

  const available = availability?.state === "available";

  return (
    <Card as="article" className="rounded-lg p-2 md:rounded-xl md:p-2.5">
      <div className="relative">
        <RoomPhoto
          room={room}
          available={available}
          eager={first}
          className="h-[190px] rounded-sm md:h-[220px] md:rounded-md"
        />
        <StatusBadge available={available} className="absolute top-2.5 left-2.5 md:top-3 md:left-3">
          {statusLabel(availability)}
        </StatusBadge>
      </div>
      <div className="flex flex-1 flex-col gap-1 px-2.5 pt-3.5 pb-2 md:gap-1.5 md:px-3 md:pt-4 md:pb-2.5">
        <h2 className="text-[19px] font-bold md:text-[21px]">{room.name}</h2>
        <p className="flex-1 text-sm text-text-muted md:text-base">
          {room.numbers} · {room.area}
        </p>
        <div className="mt-2 flex items-center justify-between gap-3 md:mt-2.5">
          <span className={cn("font-display text-xl font-medium md:text-[22px]", !available && "text-text-muted")}>
            {formatPrice(room.pricePerNight)}
          </span>
          {available ? (
            <Button href={BOOK_HREF} variant="inverse" className="px-[18px]">
              Выбрать
            </Button>
          ) : (
            <Button href={BOOK_HREF} variant="outline" className="px-4 md:px-[18px]">
              Другие даты
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}
