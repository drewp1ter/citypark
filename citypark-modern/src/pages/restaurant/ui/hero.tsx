import Image from "next/image";
import { Breadcrumbs, Button, Card, cardClass, AppLink, StatTile } from "@/shared/ui";
import { facts } from "../model/content";

const panorama = (
  <svg
    width={28}
    height={28}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={1.6}
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
  >
    <circle cx="12" cy="12" r="9" />
    <ellipse cx="12" cy="12" rx="4" ry="9" />
    <path d="M3 12h18" />
  </svg>
);

/** Intro card, photos and facts. On mobile: the intro, a photo rail and two small tiles. */
export function Hero() {
  return (
    <section className="mx-auto w-full max-w-page px-3 pt-4 md:px-6 md:pt-8">
      <div className="px-1.5 md:hidden">
        <Breadcrumbs items={[{ label: "Главная", href: "/" }, { label: "Ресторан" }]} />
      </div>
      <div className="hidden md:block">
        <Breadcrumbs items={[{ label: "Главная", href: "/" }, { label: "Ресторан и летнее кафе" }]} />
      </div>

      <div className="flex flex-col gap-2 md:grid md:grid-cols-2 md:gap-3 lg:grid-cols-4">
        <div className="flex flex-col gap-3.5 rounded-lg border border-border bg-surface p-6 md:col-span-2 md:justify-between md:gap-6 md:rounded-xl md:p-10 lg:row-span-2">
          <h1 className="heading text-[34px] md:text-[58px]">
            Ресторан
            <span className="max-md:hidden">
              <br />и летнее
              <br />
              кафе
            </span>
            <span className="md:hidden"> и летнее кафе</span>
          </h1>
          <p className="text-text-muted md:text-lg">
            <span className="md:hidden">Зал под звёздным небом и терраса у Городского парка.</span>
            <span className="max-md:hidden">
              Зал под звёздным небом с современным звуком и застеклённая охлаждаемая терраса у Городского парка.
            </span>
          </p>
          <div className="flex flex-wrap gap-2 max-md:hidden">
            <Button href="#booking" size="lg">
              Забронировать столик
            </Button>
            <Button href="/menu" variant="outline" size="lg">
              Меню
            </Button>
          </div>
        </div>

        {/* Mobile: photo rail */}
        <div
          role="group"
          aria-label="Фотографии"
          tabIndex={0}
          className="-mx-3 flex gap-2 overflow-x-auto px-3 md:hidden"
        >
          {[
            { src: "/images/restaurant-hall.jpg", alt: "Зал ресторана" },
            { src: "/images/summer-cafe-terrace.jpg", alt: "Терраса" },
            { src: "/images/glazed-terrace.jpg", alt: "Застеклённая терраса" },
          ].map((photo, index) => (
            <div key={photo.src} className="relative h-[220px] w-[300px] shrink-0 overflow-hidden rounded-lg">
              <Image
                src={photo.src}
                alt={photo.alt}
                fill
                sizes="300px"
                loading={index === 0 ? "eager" : undefined}
                fetchPriority={index === 0 ? "high" : undefined}
                className="object-cover"
              />
            </div>
          ))}
        </div>

        {/* Desktop: photos and the panorama tile */}
        <div className="relative h-[260px] overflow-hidden rounded-xl max-md:hidden md:col-span-2">
          <Image
            src="/images/restaurant-hall.jpg"
            alt="Зал ресторана"
            fill
            fetchPriority="high"
            sizes="(min-width: 1280px) 620px, 50vw"
            className="object-cover"
          />
        </div>
        <div className="relative h-[220px] overflow-hidden rounded-xl max-md:hidden">
          <Image
            src="/images/glazed-terrace.jpg"
            alt="Терраса"
            fill
            sizes="(min-width: 1280px) 300px, 25vw"
            className="object-cover"
          />
        </div>
        <AppLink
          href="#tour"
          className={cardClass("inverse", "h-[220px] justify-between rounded-xl p-6 max-md:hidden")}
        >
          <span className="text-accent">{panorama}</span>
          <span className="font-display text-xl leading-[1.15] font-medium">3D-панорама ↗</span>
        </AppLink>

        {/* Mobile: two tiles */}
        <div className="grid grid-cols-2 gap-2 md:hidden">
          <Card className="rounded-sm px-4 py-3.5">
            <span className="font-display text-xl font-medium">50–60</span>
            <span className="text-sm text-text-muted">гостей</span>
          </Card>
          <AppLink href="#tour" className={cardClass("inverse", "rounded-sm px-4 py-3.5")}>
            <span className="font-display text-xl font-medium">3D ↗</span>
            <span className="text-sm text-inverse-muted">панорама зала</span>
          </AppLink>
        </div>
      </div>

      <ul className="mt-3 grid gap-3 max-md:hidden md:grid-cols-2 lg:grid-cols-4">
        {facts.map((fact) => (
          <StatTile key={fact.value} value={fact.value} caption={fact.caption} />
        ))}
      </ul>
    </section>
  );
}
