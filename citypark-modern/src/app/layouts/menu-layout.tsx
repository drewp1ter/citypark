import type { ReactNode } from "react";
import { getMenu } from "@/entities/dish";
import { CartBar } from "@/widgets/cart-summary";
import { SiteFooter } from "@/widgets/site-footer";
import { SiteHeader } from "@/widgets/site-header";

/**
 * Frame of the menu page: header with the cart total, a wide footer, and the sticky cart bar after the footer
 * (below 1024 px; from 1024 px the cart is a column next to the dishes).
 */
export async function MenuLayout({ children }: { children: ReactNode }) {
  // A layout error would not reach the page's error.tsx, so the page throws into its own boundary instead.
  const { dishes } = await getMenu().catch((error: unknown) => {
    console.error(error);
    return { dishes: [] };
  });

  return (
    <div className="flex min-h-dvh flex-col">
      <SiteHeader cartDishes={dishes} />
      <div id="content" tabIndex={-1} className="flex-1 outline-none">
        {children}
      </div>
      <SiteFooter wide />
      <CartBar dishes={dishes} />
    </div>
  );
}
