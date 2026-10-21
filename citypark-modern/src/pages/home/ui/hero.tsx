import Image from "next/image";
import { delivery } from "@/entities/dish";
import { site } from "@/shared/config";
import { formatPrice } from "@/shared/lib";
import { AppLink, Badge, StatTile } from "@/shared/ui";

type TileProps = {
  href: string;
  image: string;
  alt: string;
  /** Desktop label (title + caption) and the shorter mobile one. */
  title: { desktop: string; mobile: string };
  caption: { desktop: string; mobile: string };
  className?: string;
  priority?: boolean;
};

/** Photo tile with a white label at the bottom. The label is smaller on mobile and has an arrow circle on desktop. */
function PhotoTile({ href, image, alt, title, caption, className, priority }: TileProps) {
  return (
    <AppLink
      href={href}
      className={`relative block h-[200px] overflow-hidden rounded-md md:min-h-[300px] md:rounded-xl ${className ?? ""}`}
    >
      <Image
        src={image}
        alt={alt}
        fill
        loading={priority ? "eager" : undefined}
        fetchPriority={priority ? "high" : undefined}
        sizes="(min-width: 1280px) 410px, (min-width: 768px) 50vw, 50vw"
        className="object-cover"
      />
      <span className="absolute inset-x-2 bottom-2 flex items-center justify-between gap-2 rounded-xs bg-surface px-3 py-2.5 md:inset-x-3.5 md:bottom-3.5 md:rounded-sm md:px-4 md:py-3.5">
        <span className="flex flex-col">
          <strong className="text-[15px] md:text-base md:font-bold">
            <span className="md:hidden">{title.mobile}</span>
            <span className="hidden md:inline">{title.desktop}</span>
          </strong>
          <span className="text-[13px] text-text-muted md:text-sm">
            <span className="md:hidden">{caption.mobile}</span>
            <span className="hidden md:inline">{caption.desktop}</span>
          </span>
        </span>
        <span
          aria-hidden="true"
          className="hidden size-9 items-center justify-center rounded-full bg-inverse-bg text-inverse-text md:inline-flex"
        >
          ↗
        </span>
      </span>
    </AppLink>
  );
}

const { phones, address } = site;
const [phoneHead, phoneTail] = ["8 918", "311-97-91"];

/** Bento grid: intro card, three photo tiles, phone card, then the facts. */
export function Hero() {
  return (
    <section className="mx-auto w-full max-w-page px-3 pt-4 md:px-6 md:pt-8">
      <div className="grid grid-cols-2 gap-2 md:gap-3 lg:grid-cols-3">
        <div className="col-span-2 flex flex-col justify-between gap-[18px] rounded-lg border border-border bg-surface p-6 md:min-h-[380px] md:gap-7 md:rounded-xl md:p-10">
          <Badge dot="md">
            <span className="hidden md:inline">{address.city} · </span>
            {address.street}
          </Badge>
          <h1 className="heading text-[40px] md:text-[length:var(--text-hero)]">
            Отель.
            <br />
            Ресторан.
            <br />
            Доставка.
          </h1>
          <p className="max-w-[520px] text-text-muted md:text-lg">
            <span className="md:hidden">Всё в одном месте — у Городского парка Белореченска.</span>
            <span className="hidden md:inline">
              Всё в одном месте — у Городского парка. Европейская и японская кухня, 12 номеров «Люкс».
            </span>
          </p>
        </div>

        <PhotoTile
          href="/rooms"
          image="/images/hotel-building.jpg"
          alt="Здание отеля"
          title={{ mobile: "Отель ↗", desktop: "Отель-люкс" }}
          caption={{ mobile: "от 4 000 ₽", desktop: "от 4 000 ₽ / ночь" }}
          className="lg:min-h-[380px]"
          priority
        />
        <PhotoTile
          href="/restaurant"
          image="/images/restaurant-hall.jpg"
          alt="Зал ресторана"
          title={{ mobile: "Ресторан ↗", desktop: "Ресторан и терраса" }}
          caption={{ mobile: "50–60 гостей", desktop: "зал на 50–60 гостей" }}
        />
        <PhotoTile
          href="/menu"
          image="/images/delivery-pasta.jpg"
          alt="Паста с креветками"
          title={{ mobile: "Доставка ↗", desktop: "Доставка" }}
          caption={{
            mobile: `0 ₽ от ${formatPrice(delivery.freeFrom)}`,
            desktop: `бесплатно от ${formatPrice(delivery.freeFrom)}`,
          }}
        />

        {/* Phone card: one big link on mobile, number + hotel number on desktop */}
        <AppLink
          href={phones.restaurant.href}
          className="flex h-[200px] flex-col justify-between rounded-md bg-inverse-bg p-4 text-inverse-text md:hidden"
        >
          <span className="text-[13px] text-inverse-muted">Позвонить</span>
          <span className="font-display text-[19px] leading-[1.2] font-medium">
            {phoneHead}
            <br />
            {phoneTail}
          </span>
        </AppLink>
        <div className="hidden min-h-[300px] flex-col justify-between gap-[18px] rounded-xl bg-inverse-bg p-8 text-inverse-text md:flex">
          <p className="text-sm text-inverse-muted">Столик, доставка, банкет</p>
          <AppLink
            href={phones.restaurant.href}
            className="font-display text-[28px] leading-[1.15] font-medium tracking-[-0.02em]"
          >
            {phoneHead}
            <br />
            {phoneTail}
          </AppLink>
          <p className="text-sm text-inverse-muted">
            Отель:{" "}
            <AppLink href={phones.hotel.href} className="tap-area text-inverse-text">
              {phones.hotel.display}
            </AppLink>
          </p>
        </div>
      </div>

      <ul className="mt-2 grid grid-cols-2 gap-2 md:mt-3 md:gap-3 lg:grid-cols-4">
        <StatTile
          value="12"
          caption={
            <>
              номеров<span className="hidden md:inline">, 19–45 м²</span>
            </>
          }
        />
        <StatTile
          value="24/7"
          caption={
            <>
              заезд<span className="hidden md:inline"> в отель</span>
            </>
          }
        />
        <StatTile className="hidden md:block" value="0 ₽" caption="Wi-Fi и парковка" />
        <StatTile className="hidden md:block" value="30 мин" caption="готовим доставку" />
      </ul>
    </section>
  );
}
