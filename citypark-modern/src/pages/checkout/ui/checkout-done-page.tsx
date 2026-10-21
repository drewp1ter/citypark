import { delivery } from "@/entities/dish";
import { site } from "@/shared/config";
import { Button, cardClass } from "@/shared/ui";
import { CheckoutHeader } from "./checkout-header";
import { ClearCart } from "./clear-cart";
import { TitleRow } from "./title-row";

/** Step 3, "Готово". The order action redirects here after the order has been sent. */
export function CheckoutDonePage() {
  return (
    <>
      <CheckoutHeader step={3} />
      <main className="mx-auto w-full max-w-narrow-page px-3 pt-2 pb-6 md:px-6 md:pt-10 md:pb-10">
        <ClearCart />
        <TitleRow step={3} title="Заказ принят" />
        <div className={cardClass("surface", "items-start gap-3 rounded-lg p-6 md:rounded-xl md:p-10")}>
          <h2 className="heading text-2xl md:text-[32px]">Спасибо за заказ!</h2>
          <p className="max-w-xl text-text-muted">
            Мы перезвоним по указанному номеру, чтобы подтвердить заказ. Доставляем {delivery.hours}. Если что-то
            изменилось, позвоните: <span className="whitespace-nowrap text-text">{site.phones.restaurant.display}</span>
            .
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            <Button href="/menu" size="xl" variant="inverse">
              Вернуться в меню
            </Button>
            <Button href="/" size="xl" variant="outline">
              На главную
            </Button>
          </div>
        </div>
      </main>
    </>
  );
}
