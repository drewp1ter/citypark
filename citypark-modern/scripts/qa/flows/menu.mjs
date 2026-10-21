// Flow check: menu. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/menu.mjs   (server must run on BASE_URL, default http://localhost:3111)
import { chromium } from "playwright";
const base = process.env.BASE_URL ?? "http://localhost:3111";
const browser = await chromium.launch();
const out = (k, v) => console.log(k.padEnd(56), v);
const txt = async (loc) =>
  (
    await loc
      .first()
      .innerText()
      .catch(() => "—")
  ).replace(/\s+/g, " ");

async function open(path, viewport, cart) {
  const ctx = await browser.newContext({ viewport });
  if (cart) await ctx.addInitScript((c) => localStorage.setItem("citypark-cart", JSON.stringify(c)), cart);
  const p = await ctx.newPage();
  const problems = [];
  p.on("pageerror", (e) => problems.push(e.message.slice(0, 120)));
  await p.goto(base + path);
  await p.waitForTimeout(500);
  return Object.assign(p, { problems });
}
const D = { width: 1440, height: 900 },
  M = { width: 390, height: 844 };

// ---- desktop
let p = await open("/menu", D);
out(
  "h1 / h2 / dish cards",
  `${await txt(p.locator("h1"))} / ${await txt(p.locator("main h2").first())} / ${await p.locator("article").count()}`,
);
out("header: cart pill (empty cart)", await txt(p.getByRole("link", { name: /^Корзина/ }).first()));
out("cart panel (empty)", await txt(p.locator("aside#cart")));
out("CartBar hidden at 1440", String((await p.locator("a[href='/checkout']:visible").count()) === 0 || "panel only"));

await p.getByRole("button", { name: "Добавить в корзину: салат свекольный" }).click();
await p.getByRole("button", { name: "Добавить в корзину: крем-суп из тыквы" }).click();
await p.getByRole("button", { name: "Добавить порцию" }).last().click();
await p.waitForTimeout(200);
out("pill after 1 salad + 2 soups", await txt(p.getByRole("link", { name: /^Корзина/ }).first()));
out("panel", await txt(p.locator("aside#cart")));

// size switch on the pie
const pie = p.locator("article", { hasText: "Пирог с капустой" });
await pie.getByRole("button", { name: "1 кг" }).click();
out(
  "pie 1 kg: price + add disabled",
  `${await txt(pie.locator("span.font-display"))} / disabled=${await pie
    .getByRole("button", { name: /Добавить в корзину/ })
    .first()
    .isDisabled()}`,
);
await pie.getByRole("button", { name: "500 г" }).click();
await pie
  .getByRole("button", { name: /Добавить в корзину/ })
  .first()
  .click();
out("pie 500 g added -> panel line", await txt(p.locator("aside#cart li", { hasText: "Пирог" })));
out("cart in localStorage", await p.evaluate(() => localStorage.getItem("citypark-cart")));

// categories (links, server-side)
await p.getByRole("link", { name: "Стейки", exact: true }).click();
await p.waitForURL("**/menu?category=steaks");
await p.waitForTimeout(300);
out(
  "category Стейки: url / h2 / message",
  `${p.url().replace(base, "")} / ${await txt(p.locator("main h2").first())} / ${await txt(p.getByRole("status"))}`,
);
out("selected chip", await txt(p.locator("nav[aria-label='Разделы меню'] a[aria-current=true]")));
await p.getByRole("link", { name: "Постное", exact: true }).click();
await p.waitForURL("**/menu");
await p.waitForTimeout(300);
out("back to Постное: dishes", String(await p.locator("article").count()));
out("cart survives navigation", await txt(p.getByRole("link", { name: /^Корзина/ }).first()));

// search across the whole menu
const search = p.getByRole("searchbox", { name: "Поиск по меню" });
await search.fill("суп");
out("search 'суп'", `${await txt(p.locator("main h2").first())} / ${await p.locator("article").count()} card`);
await search.fill("ТОФУ");
out(
  "search 'ТОФУ' (composition + case)",
  `${await txt(p.locator("main h2").first())} / ${await p.locator("article h3").allInnerTexts()}`,
);
await search.fill("пицца-ролл");
out("search no match", await txt(p.getByRole("status")));
await p.getByRole("link", { name: "Роллы", exact: true }).click();
await p.waitForTimeout(500);
out("search text kept when category changes", await search.inputValue());
await search.fill("");

// sticky toolbar
await p.evaluate(() => window.scrollTo(0, 700));
await p.waitForTimeout(200);
out(
  "sticky: search + chips top (px)",
  JSON.stringify(
    await p.evaluate(() =>
      [
        document.querySelector("input[type=search]").closest("div[class*=sticky]").getBoundingClientRect().top,
        document.querySelector("nav[aria-label='Разделы меню']").getBoundingClientRect().top,
      ].map(Math.round),
    ),
  ),
);
out(
  "sticky: cart panel top (px)",
  String(Math.round(await p.locator("aside#cart").evaluate((e) => e.getBoundingClientRect().top))),
);
out(
  "unknown category falls back",
  await (async () => {
    const q = await open("/menu?category=nope", D);
    return `${await txt(q.locator("main h2").first())} / ${await q.locator("article").count()}`;
  })(),
);

await p.locator("aside#cart").getByRole("link", { name: "Оформить заказ" }).click();
await p.waitForURL("**/checkout");
await p.waitForTimeout(500);
out("Оформить заказ -> checkout summary", (await txt(p.locator("aside[aria-label='Ваш заказ']"))).slice(0, 120));
out("page errors (desktop)", JSON.stringify(p.problems));

// ---- mobile
const m = await open("/menu", M, [
  { dishId: "beet-salad", variantId: "default", quantity: 1 },
  { dishId: "pumpkin-soup", variantId: "default", quantity: 2 },
  { dishId: "mushroom-risotto", variantId: "default", quantity: 1 },
]);
out("mobile: cart bar", await txt(m.locator("a[href='/checkout']").last()));
out("mobile: aside hidden", String(await m.locator("aside#cart").isVisible()));
out("mobile: card pills", (await m.locator("article").first().innerText()).replace(/\s+/g, " "));
await m.getByRole("button", { name: "Добавить в корзину: салат овощной" }).click();
out("mobile: add -> bar", await txt(m.locator("a[href='/checkout']").last()));
out("mobile: overflow", String(await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)));
await m.evaluate(() => window.scrollTo(0, 600));
await m.waitForTimeout(200);
out(
  "mobile: chips sticky top (px)",
  String(Math.round(await m.locator("nav[aria-label='Разделы меню']").evaluate((e) => e.getBoundingClientRect().top))),
);
// tablet
const t = await open("/menu", { width: 900, height: 900 }, [
  { dishId: "beet-salad", variantId: "default", quantity: 1 },
]);
out(
  "tablet 900: cart bar visible / aside hidden / cols",
  `${await t.locator("a[href='/checkout']").last().isVisible()} / ${await t.locator("aside#cart").isVisible()} / ${await t.evaluate(() => getComputedStyle(document.querySelector("article").parentElement).gridTemplateColumns.split(" ").length)}`,
);
await browser.close();
