// Flow check: menu. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/menu.mjs   (server must run on BASE_URL, default http://localhost:3111)
import { chromium } from "playwright";
import { seedCart } from "../menu-cart.mjs";
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

const addButton = p.getByRole("button", { name: /^Добавить в корзину: /, disabled: false });
const [firstAdd, secondAdd] = await addButton.evaluateAll((els) => els.slice(0, 2).map((e) => e.getAttribute("aria-label")));
await p.getByRole("button", { name: firstAdd, exact: true }).click();
await p.getByRole("button", { name: secondAdd, exact: true }).click();
await p.locator("article").getByRole("button", { name: "Добавить порцию" }).last().click();
await p.waitForTimeout(200);
out("pill after 1 + 2 portions", await txt(p.getByRole("link", { name: /^Корзина/ }).first()));
out("panel", await txt(p.locator("aside#cart")));

// steppers inside the panel
const panel = p.locator("aside#cart");
const pill = () => txt(p.getByRole("link", { name: /^Корзина/ }).first());
const panelTotal = () => txt(panel.locator("div", { hasText: /^Итого/ }).last());
const beforePlus = `${await panelTotal()} / ${await pill()}`;
await panel.getByRole("button", { name: "Добавить порцию" }).first().click();
await p.waitForTimeout(200);
out("panel + -> total / header pill", `${beforePlus} -> ${await panelTotal()} / ${await pill()}`);
const linesBefore = await panel.locator("li").count();
const removeButton = panel.locator("li").first().getByRole("button", { name: "Убрать порцию" });
while ((await panel.locator("li").count()) === linesBefore) {
  await removeButton.click();
  await p.waitForTimeout(100);
}
out("panel − -> lines", `${linesBefore} -> ${await panel.locator("li").count()}`);
out("panel − -> dish card shows the round + again", String((await p.getByRole("button", { name: firstAdd, exact: true }).count()) === 1));

// backend dishes have one size, so there is no size switch
out("no size switch", String((await p.getByRole("group", { name: "Размер" }).count()) === 0));
out("cart in localStorage", await p.evaluate(() => localStorage.getItem("citypark-cart")));

// categories (links, server-side)
const chips = p.locator("nav[aria-label='Разделы меню'] a");
const firstChip = (await chips.first().innerText()).trim();
const secondHref = await chips.nth(1).getAttribute("href");
await chips.nth(1).click();
await p.waitForURL("**" + secondHref);
await p.waitForTimeout(300);
out(
  "second category: url / h2 / message",
  `${p.url().replace(base, "")} / ${await txt(p.locator("main h2").first())} / ${await txt(p.getByRole("status"))}`,
);
out("selected chip", await txt(p.locator("nav[aria-label='Разделы меню'] a[aria-current=true]")));
await p.getByRole("link", { name: firstChip, exact: true }).click();
await p.waitForURL("**/menu");
await p.waitForTimeout(300);
out("back to the first category: dishes", String(await p.locator("article").count()));
out("cart survives navigation", await txt(p.getByRole("link", { name: /^Корзина/ }).first()));

// search across the whole menu
const search = p.getByRole("searchbox", { name: "Поиск по меню" });
const cardsBeforeSearch = await p.locator("article").count();
const word = firstAdd.replace(/^Добавить в корзину: /, "").split(/\s+/)[0];
await search.fill(word);
out(`search '${word}'`, `${await txt(p.locator("main h2").first())} / ${await p.locator("article").count()} card`);
await search.fill(word.toUpperCase());
out(
  `search '${word.toUpperCase()}' (case)`,
  `${await txt(p.locator("main h2").first())} / ${await p.locator("article h3").allInnerTexts()}`,
);
await search.fill("пицца-ролл");
out("search no match", await txt(p.getByRole("status")));
out("search: clear button shown", String((await p.getByRole("button", { name: "Очистить поле" }).count()) === 1));
await p.getByRole("button", { name: "Очистить поле" }).click();
out(
  `search: after clear (value / cards / focus), expected "" / ${cardsBeforeSearch} / true`,
  `${await search.inputValue()} / ${await p.locator("article").count()} / ${await p.evaluate(() => document.activeElement.type === "search")}`,
);
out("search: clear button hidden when empty", String((await p.getByRole("button", { name: "Очистить поле" }).count()) > 0));
await search.fill("пицца-ролл");
await chips.nth(1).click();
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
  "toolbar: header bottom / chips top (px)",
  JSON.stringify(
    await p.evaluate(() => [
      document.querySelector("header").getBoundingClientRect().bottom,
      document.querySelector("nav[aria-label='Разделы меню']").getBoundingClientRect().top,
    ].map(Math.round)),
  ),
);
const scroller = p.locator("nav[aria-label='Разделы меню'] > div > div");
const arrowLeft = p.getByRole("button", { name: "Прокрутить разделы влево", includeHidden: true });
const arrowRight = p.getByRole("button", { name: "Прокрутить разделы вправо", includeHidden: true });
out("chips: scrollbar-width", await scroller.evaluate((e) => getComputedStyle(e).scrollbarWidth));
out(
  "chips: arrows at start",
  `${await arrowLeft.isVisible()} / ${await arrowRight.isVisible()} (scrollWidth > clientWidth: ${await scroller.evaluate((e) => e.scrollWidth > e.clientWidth)})`,
);
if (await arrowRight.isVisible()) {
  await arrowRight.click();
  await p.waitForTimeout(600);
  out("chips: after ›", `${await scroller.evaluate((e) => e.scrollLeft > 0)} / ${await arrowLeft.isVisible()}`);
}
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
const m = await open("/menu", M, await seedCart([1, 2, 1]));
const bar = m.locator("nav[aria-label='Корзина'] button");
out("mobile: cart bar", await txt(bar));
out("mobile: aside hidden", String(await m.locator("aside#cart").isVisible()));
out("mobile: card pills", (await m.locator("article").first().innerText()).replace(/\s+/g, " "));
await m.getByRole("button", { name: /^Добавить в корзину: /, disabled: false }).first().click();
out("mobile: add -> bar", await txt(bar));
await bar.click();
const sheet = m.locator("dialog[open]");
out("mobile: sheet", (await txt(sheet)).replace(/\s+/g, " "));
const sheetTotal = () => txt(sheet.locator("div", { hasText: /^Итого/ }).last());
const before = await sheetTotal();
await sheet.getByRole("button", { name: "Добавить порцию" }).first().click();
out("mobile: sheet + -> total / bar", `${before} -> ${await sheetTotal()} / ${await txt(bar)}`);
await m.keyboard.press("Escape");
out(
  "mobile: Esc -> sheet open / focus on bar",
  `${await m.locator("dialog[open]").count()} / ${await bar.evaluate((e) => e === document.activeElement)}`,
);
out("mobile: overflow", String(await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)));
await m.evaluate(() => window.scrollTo(0, 600));
await m.waitForTimeout(200);
out(
  "mobile: chips sticky top (px)",
  String(Math.round(await m.locator("nav[aria-label='Разделы меню']").evaluate((e) => e.getBoundingClientRect().top))),
);
out(
  "mobile: chip arrows",
  `${await m.getByRole("button", { name: "Прокрутить разделы влево", includeHidden: true }).isVisible()} / ${await m.getByRole("button", { name: "Прокрутить разделы вправо", includeHidden: true }).isVisible()}`,
);
// description on the card (needs a backend with synced descriptions)
const apiBase = process.env.MENU_API_URL ?? "http://localhost:3000/api/v1";
const products = await fetch(`${apiBase}/products?pageSize=500`)
  .then((r) => r.json())
  .catch(() => []);
const described = products.find((x) => x.isAvailableForOrder && x.description?.trim());
if (!described) {
  out("description on card", "skipped: no product has a description");
} else {
  for (const [label, viewport] of [["desktop", D], ["mobile", M]]) {
    const q = await open(`/menu?category=${described.categoryId}`, viewport);
    const card = q.locator("article", { has: q.locator("h3", { hasText: described.name }) }).first();
    const desc = card.locator("h3 + p");
    const lines = await desc.evaluate((el) => el.clientHeight <= 2 * parseFloat(getComputedStyle(el).lineHeight) + 1).catch(() => "—");
    out(`description on card ${label}`, `${(await txt(desc)).slice(0, 60)} / lines <= 2: ${lines}`);
  }
}
// tablet
const t = await open("/menu", { width: 900, height: 900 }, await seedCart([1]));
out(
  "tablet 900: cart bar visible / aside hidden / cols",
  `${await t.locator("nav[aria-label='Корзина'] button").isVisible()} / ${await t.locator("aside#cart").isVisible()} / ${await t.evaluate(() => getComputedStyle(document.querySelector("article").parentElement).gridTemplateColumns.split(" ").length)}`,
);
// tapping «Оформить заказ» in the sheet goes to the checkout
const c = await open("/menu", M, await seedCart([1, 2, 1]));
await c.locator("nav[aria-label='Корзина'] button").click();
await c.locator("dialog[open]").getByRole("link", { name: "Оформить заказ" }).click();
await c.waitForURL("**/checkout");
out("mobile: sheet -> checkout", new URL(c.url()).pathname);
await browser.close();
