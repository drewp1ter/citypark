import { delivery, getMenu } from "@/entities/dish";
import { formatPrice } from "@/shared/lib";
import { CartPanel } from "@/widgets/cart-summary";
import { MenuCatalog } from "./menu-catalog";

const pill = "rounded-pill border border-border bg-surface px-3 py-1.5 text-sm md:px-4 md:py-2 md:text-base";

/** Delivery menu: facts, search and categories, dishes, and the cart column. */
export async function MenuPage({ categoryId }: { categoryId?: string }) {
  const menu = await getMenu();

  return (
    <main className="mx-auto w-full max-w-wide-page px-3 pt-2 md:px-6 md:pt-8">
      <div className="flex flex-wrap items-end justify-between gap-x-5 gap-y-3">
        <h1 className="heading mx-1.5 mt-2 text-4xl md:mx-0 md:mt-0 md:text-[length:var(--text-h1)]">Доставка</h1>
        <ul className="flex flex-wrap gap-1.5 md:gap-2">
          <li className="rounded-pill bg-accent px-3 py-1.5 text-sm font-semibold text-on-accent md:px-4 md:py-2 md:text-base">
            0 ₽ от {formatPrice(delivery.freeFrom)}
          </li>
          <li className={pill}>{delivery.hours}</li>
          <li className={pill}>
            {delivery.cooking}
            <span className="hidden md:inline"> готовим</span>
          </li>
          <li className={`${pill} hidden md:block`}>{delivery.outskirtsFee}</li>
        </ul>
      </div>

      <MenuCatalog
        categories={menu.categories}
        dishes={menu.dishes}
        categoryId={categoryId}
        aside={<CartPanel dishes={menu.dishes} />}
      />
    </main>
  );
}
