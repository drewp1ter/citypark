import type { ReactNode } from "react";
import { SiteLayout } from "./site-layout";

/** Page frame without the mobile quick-actions bar (contacts). */
export function PlainLayout({ children }: { children: ReactNode }) {
  return <SiteLayout quickActions={false}>{children}</SiteLayout>;
}
