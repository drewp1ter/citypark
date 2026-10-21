// Flow check: checkout-and-contacts. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/checkout-and-contacts.mjs   (server must run on BASE_URL, default http://localhost:3111)
import { chromium } from "playwright";
const base = process.env.BASE_URL ?? "http://localhost:3111";
const browser = await chromium.launch();
const out = (k, v) => console.log(k.padEnd(58), v);
const CART = [
  { dishId: "beet-salad", variantId: "default", quantity: 1 },
  { dishId: "pumpkin-soup", variantId: "default", quantity: 2 },
  { dishId: "mushroom-risotto", variantId: "default", quantity: 1 },
];
async function pageWithCart(cart = CART, viewport = { width: 1440, height: 1000 }) {
  const ctx = await browser.newContext({ viewport });
  if (cart) await ctx.addInitScript((c) => localStorage.setItem("citypark-cart", JSON.stringify(c)), cart);
  return ctx.newPage();
}
const txt = async (loc) =>
  (
    await loc
      .first()
      .innerText()
      .catch(() => "—")
  ).replace(/\s+/g, " ");

// ---------- checkout
let p = await pageWithCart();
await p.goto(base + "/checkout");
await p.waitForTimeout(500);
out("summary: lines + total", await txt(p.locator("aside[aria-label='Ваш заказ']")));
await p.getByRole("button", { name: "Подтвердить заказ" }).click();
await p.waitForTimeout(700);
out("empty submit -> alerts", JSON.stringify((await p.getByRole("alert").allInnerTexts()).filter(Boolean)));
await p.getByLabel("Имя").fill("Анна");
await p.getByLabel("Улица и дом").fill("Гоголя 10");
await p.getByRole("button", { name: "Подтвердить заказ" }).click();
await p.waitForTimeout(700);
out(
  "typed name + street kept after error",
  `${await p.getByLabel("Имя").inputValue()} | ${await p.getByLabel("Улица и дом").inputValue()}`,
);
out("remaining alerts", JSON.stringify((await p.getByRole("alert").allInnerTexts()).filter(Boolean)));

// variants: outskirts, pickup, scheduled
await p.getByLabel("Населённый пункт").selectOption("outskirts");
out(
  "outskirts: delivery line + note",
  (await txt(p.locator("aside li", { hasText: "Доставка" }))) + " || " + (await txt(p.getByText("не входит в сумму"))),
);
await p.getByLabel("Когда").selectOption("scheduled");
await p.getByLabel("Время", { exact: true }).fill("23:45");
await p.getByRole("button", { name: "Подтвердить заказ" }).click();
await p.waitForTimeout(700);
out("scheduled 23:45 -> error", JSON.stringify((await p.getByRole("alert").allInnerTexts()).filter(Boolean)));
await p.getByText("Самовывоз").click();
out(
  "pickup: address fields hidden",
  String((await p.getByLabel("Улица и дом").count()) === 0 && (await p.getByLabel("Населённый пункт").count()) === 0),
);
out("pickup: delivery line", await txt(p.locator("aside li", { hasText: "Доставка" })));

// tampered cart -> server refuses
const t = await pageWithCart([{ dishId: "ghost-dish", variantId: "x", quantity: 1 }]);
await t.goto(base + "/checkout");
await t.waitForTimeout(500);
out("unknown dish in cart -> priced out, empty state", await txt(t.getByText("В корзине пока пусто")));
// pie 1 kg has no price: the client skips it, server too
const t2 = await pageWithCart([{ dishId: "cabbage-pie", variantId: "1kg", quantity: 1 }]);
await t2.goto(base + "/checkout");
await t2.waitForTimeout(500);
out("unpriced 1 kg pie -> empty state", await txt(t2.getByText("В корзине пока пусто")));
// server-side check without the browser's own filtering
const res = await fetch(base + "/checkout", { method: "GET" });
out("GET /checkout", res.status);

// valid order
await p.getByText("Доставка", { exact: true }).first().click();
await p.getByLabel("Улица и дом").fill("Гоголя 10");
await p.getByLabel("Когда").selectOption("asap");
await p.getByLabel("Телефон").fill("");
await p.getByLabel("Телефон").pressSequentially("89183119791");
await p.getByLabel(/Согласен/).check();
await p.getByRole("button", { name: "Подтвердить заказ" }).click();
await p.waitForURL("**/checkout/done");
await p.waitForTimeout(500);
out("after valid order: url", p.url().replace(base, ""));
out("done page h1 + step", `${await txt(p.locator("h1"))} | ${await txt(p.locator("ol li[aria-current=step]"))}`);
out("cart cleared", String(await p.evaluate(() => localStorage.getItem("citypark-cart"))));

// empty cart
const e = await pageWithCart(null);
await e.goto(base + "/checkout");
await e.waitForTimeout(500);
out(
  "empty cart state + link",
  `${await txt(e.getByText("В корзине пока пусто"))} -> ${await e.getByRole("link", { name: "Перейти в меню" }).getAttribute("href")}`,
);
out("empty cart: no form / no submit", String(await e.locator("form#checkout").count()));

// ---------- contact form
const c = await (await browser.newContext({ viewport: { width: 1440, height: 1000 } })).newPage();
await c.goto(base + "/contacts");
await c.waitForTimeout(500);
await c.getByRole("button", { name: "Отправить", exact: true }).click();
await c.waitForTimeout(700);
out("empty contact form -> alerts", JSON.stringify((await c.getByRole("alert").allInnerTexts()).filter(Boolean)));
await c.getByLabel("Имя").fill("Иван");
await c.getByLabel("Телефон или e-mail").fill("не телефон");
await c.getByLabel("Сообщение").fill("Здравствуйте, есть вопрос");
await c.getByRole("button", { name: "Отправить", exact: true }).click();
await c.waitForTimeout(700);
out(
  "bad contact -> error, message kept",
  `${await txt(c.getByRole("alert").filter({ hasText: /\S/ }))} | ${await c.getByLabel("Сообщение").inputValue()}`,
);
await c.getByText("Банкет", { exact: true }).click();
await c.getByLabel("Телефон или e-mail").fill("ivan@example.com");
await c.getByRole("button", { name: "Отправить", exact: true }).click();
await c.waitForTimeout(800);
out("valid contact -> success", await txt(c.getByRole("status")));
// honeypot
const h = await (await browser.newContext()).newPage();
await h.goto(base + "/contacts");
await h.waitForTimeout(400);
await h.getByLabel("Имя").fill("Бот");
await h.getByLabel("Телефон или e-mail").fill("bot@spam.com");
await h.getByLabel("Сообщение").fill("buy now buy now");
await h.locator("input[name=website]").fill("http://spam", { force: true });
await h.getByRole("button", { name: "Отправить", exact: true }).click();
await h.waitForTimeout(800);
out("honeypot filled -> looks like success", await txt(h.getByRole("status")));

// ---------- 404 and articles
const status = async (path) => (await fetch(base + path)).status;
out("GET /nope", await status("/nope"));
out("GET /news/unknown-slug", await status("/news/unknown-slug"));
out("GET /news/frittata", await status("/news/frittata"));
out("GET /news/new-menu-2026", await status("/news/new-menu-2026"));
const a = await browser.newPage();
await a.goto(base + "/nope");
out(
  "404 page h1 + home link",
  `${await txt(a.locator("h1"))} | ${await a.getByRole("link", { name: "На главную" }).getAttribute("href")}`,
);
await a.goto(base + "/news/frittata");
out("article h1 / meta", `${await txt(a.locator("h1"))} / ${await txt(a.locator("article p").first())}`);
await a.goto(base + "/");
await a
  .locator("a[href='/news/special-offer-september-2026']")
  .first()
  .click({ force: true })
  .catch(() => {});
await a.waitForTimeout(500);
out("home news card opens the article", a.url().replace(base, ""));
await browser.close();
