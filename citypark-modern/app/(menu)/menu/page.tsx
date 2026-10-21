import type { Metadata } from "next";
import { MenuPage } from "@/pages/menu";

export const metadata: Metadata = { title: "Доставка — Сити Парк" };

// The menu reads ?category=… on the server (and will read live prices from the POS), so it is rendered per request.
export default async function Page({ searchParams }: PageProps<"/menu">) {
  const { category } = await searchParams;
  return <MenuPage categoryId={typeof category === "string" ? category : undefined} />;
}
