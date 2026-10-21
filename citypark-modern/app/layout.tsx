import type { Metadata } from "next";
import { Onest, Unbounded } from "next/font/google";
import { ThemeProvider } from "@/app/theme";
import "@/app/styles/globals.css";

const unbounded = Unbounded({
  variable: "--font-unbounded",
  weight: ["400", "500", "600"],
  subsets: ["latin", "cyrillic"],
  display: "swap",
});

const onest = Onest({
  variable: "--font-onest",
  weight: ["400", "500", "600", "700"],
  subsets: ["latin", "cyrillic"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "Сити Парк — отель, ресторан и доставка в Белореченске",
  description: "Отель-люкс на 12 номеров, ресторан и летнее кафе, банкеты и доставка еды. Белореченск, ул. Гоголя, 61.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ru" className={`${unbounded.variable} ${onest.variable}`} suppressHydrationWarning>
      <body>
        <a
          href="#content"
          className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50 focus:rounded-pill focus:bg-inverse-bg focus:px-5 focus:py-3 focus:font-semibold focus:text-inverse-text"
        >
          Перейти к содержимому
        </a>
        <ThemeProvider>{children}</ThemeProvider>
      </body>
    </html>
  );
}
