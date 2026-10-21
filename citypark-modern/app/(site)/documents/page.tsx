import type { Metadata } from "next";
import { LegalPage } from "@/pages/legal";

export const metadata: Metadata = { title: "Документы — Сити Парк" };

export default function Page() {
  return <LegalPage title="Документы" />;
}
