import { TableBookingForm } from "@/features/book-table";
import { site } from "@/shared/config";
import { AppLink } from "@/shared/ui";

const { phones } = site;

/** Dark booking block: heading and phone numbers on the left (desktop), the form on the right. */
export function BookingSection() {
  return (
    <section id="booking" className="mx-auto w-full max-w-page scroll-mt-24 px-3 pt-12 md:px-6 md:pt-[104px] md:pb-6">
      <div className="flex flex-wrap gap-10 rounded-[30px] bg-inverse-bg p-[22px] text-inverse-text md:rounded-2xl md:p-12">
        <div className="hidden min-w-0 flex-[1_1_340px] flex-col gap-4 md:flex">
          <h2 className="heading text-[40px] leading-[1.05]">Забронировать столик</h2>
          <p className="text-inverse-muted">Перезвоним, чтобы подтвердить бронь.</p>
          <AppLink href={phones.restaurant.href} className="tap-area font-display text-[26px] font-medium">
            {phones.restaurant.display}
          </AppLink>
          <AppLink href={phones.restaurant.alt.href} className="tap-area text-lg text-inverse-muted">
            {phones.restaurant.alt.display}
          </AppLink>
        </div>
        <TableBookingForm variant="dark" className="min-w-0 flex-[1_1_520px]" />
      </div>
    </section>
  );
}
