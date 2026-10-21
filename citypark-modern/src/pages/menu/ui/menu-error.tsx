"use client";

import { useEffect } from "react";
import { site } from "@/shared/config";
import { Button } from "@/shared/ui";

/** Shown by the `error.tsx` of /menu and /checkout when the menu cannot be read from the backend. */
export function MenuError({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="mx-auto flex w-full max-w-page flex-col items-start gap-4 px-3 py-16 md:px-6 md:py-28">
      <h1 className="text-2xl font-bold md:text-3xl">Меню временно недоступно</h1>
      <p className="max-w-md text-text-muted">
        Не получилось загрузить меню. Попробуйте обновить страницу или закажите по телефону.
      </p>
      <div className="mt-2 flex flex-wrap gap-2">
        <Button size="xl" onClick={() => retry()}>
          Обновить
        </Button>
        <Button href={site.phones.restaurant.href} size="xl" variant="outline">
          {site.phones.restaurant.display}
        </Button>
      </div>
    </main>
  );
}
