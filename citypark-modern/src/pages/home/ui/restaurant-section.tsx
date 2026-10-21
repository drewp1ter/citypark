import Image from "next/image";
import { TableBookingForm } from "@/features/book-table";
import { AppLink, Button, Card, cardClass, Section } from "@/shared/ui";

const promoTitle = "font-display text-xl leading-[1.15] font-medium tracking-[-0.02em] md:text-2xl";

export function RestaurantSection() {
  return (
    <Section title="Ресторан">
      <div className="grid gap-2 md:grid-cols-2 md:gap-3 lg:grid-cols-3">
        <div className="relative h-[220px] overflow-hidden rounded-lg md:col-span-2 md:h-[420px] md:rounded-xl">
          <Image
            src="/images/summer-cafe-terrace.jpg"
            alt="Летняя терраса"
            fill
            sizes="(min-width: 1280px) 820px, (min-width: 768px) 100vw, 100vw"
            className="object-cover"
          />
        </div>

        {/* On mobile the booking form is replaced by a button to the restaurant page. */}
        <TableBookingForm className="hidden md:col-span-2 md:flex lg:col-span-1 lg:row-span-2" />

        <Card
          variant="accent"
          className="gap-1.5 rounded-lg p-5 md:min-h-[220px] md:justify-between md:gap-4 md:rounded-xl md:p-7"
        >
          <span className={promoTitle}>Банкеты на 50–60 гостей</span>
          <span className="text-[15px]">
            Свадьбы, юбилеи, корпоративы<span className="hidden md:inline">. Танцпол и профессиональный звук.</span>
          </span>
        </Card>

        <AppLink
          href="/restaurant"
          className={cardClass("surface", "hidden min-h-[220px] justify-between gap-4 rounded-xl p-7 md:flex")}
        >
          <span className={promoTitle}>3D-панорама зала</span>
          <span className="text-[15px] text-text-muted">
            Европейская и японская кухня, джаз и chill-out · Смотреть ↗
          </span>
        </AppLink>

        <Button href="/restaurant" variant="inverse" size="xl" className="md:hidden">
          Забронировать столик
        </Button>
      </div>
    </Section>
  );
}
