import { getMenu } from "@/entities/dish";
import { CheckoutForm } from "./checkout-form";
import { CheckoutHeader } from "./checkout-header";
import { TitleRow } from "./title-row";

export async function CheckoutPage() {
  const menu = await getMenu();

  return (
    <>
      <CheckoutHeader step={2} />
      <main>
        <CheckoutForm dishes={menu.dishes} intro={<TitleRow step={2} title="Оформление" />} />
      </main>
    </>
  );
}
