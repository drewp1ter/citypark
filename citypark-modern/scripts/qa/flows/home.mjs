// Flow check: home. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/home.mjs   (server must run on BASE_URL, default http://localhost:3111)
import { chromium } from "playwright";
const base = process.env.BASE_URL ?? "http://localhost:3111";
const browser = await chromium.launch();
const out = (k, v) => console.log(k.padEnd(52), v);

// ---- colours in dark theme
const dctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: "dark" });
await dctx.addInitScript(() => localStorage.setItem("theme", "dark"));
const d = await dctx.newPage();
await d.goto(base + "/");
// The dark theme's inverse surfaces must all be the design's #2A2A2F.
const inverse = await d.evaluate(() =>
  [...document.querySelectorAll(".bg-inverse-bg")].map((e) => getComputedStyle(e).backgroundColor),
);
out(
  "dark: every .bg-inverse-bg is rgb(42, 42, 47)",
  String(inverse.length > 0 && inverse.every((c) => c === "rgb(42, 42, 47)")),
);

// ---- today/tomorrow defaults
const p = await browser.newPage({ viewport: { width: 1440, height: 900 } });
await p.goto(base + "/");
await p.waitForTimeout(400);
const iso = (dt) =>
  `${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, "0")}-${String(dt.getDate()).padStart(2, "0")}`;
const now = new Date();
const tom = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
out("stay bar check-in = today", `${await p.locator("input[name=checkIn]").inputValue()}  (want ${iso(now)})`);
out("stay bar check-out = tomorrow", `${await p.locator("input[name=checkOut]").inputValue()}  (want ${iso(tom)})`);
out("booking form date = today", await p.locator("form input[name=date]").inputValue());
await p.locator("input[name=checkIn]").fill(iso(new Date(now.getFullYear(), now.getMonth(), now.getDate() + 5)));
out("check-out follows a later check-in", await p.locator("input[name=checkOut]").inputValue());
await p.getByRole("button", { name: "Проверить" }).click();
await p.waitForURL("**/rooms?**");
out("search submits to", p.url().replace(base, ""));

// ---- free rooms count
// Set known numbers on the backend first:
// curl -s -X POST localhost:3000/api/v1/hotel/rooms -H "X-Webhook-Secret: $HOTEL_WEBHOOK_SECRET" \
//   -H 'Content-Type: application/json' -d '{"rooms":{"vip":1,"singleBalcony":0,"single":2,"doubleTwin":1,"doubleEuro":0}}'
const r = await browser.newPage({ viewport: { width: 1440, height: 900 } });
await r.goto(base + "/");
const fr = r.locator("p", { has: r.locator("span.sr-only", { hasText: "Сегодня свободно" }) });
out(
  "rooms: free count tile / phrase",
  `${await fr.locator("span[aria-hidden]").first().innerText()} / ${await fr.locator("span.text-lg").innerText()}  (want 4 / свободных номера)`,
);
out("rooms: screen reader sentence", `${await fr.locator("span.sr-only").innerText()}  (want Сегодня свободно 4 номера из 12)`);

// ---- cart
const c = await browser.newPage({ viewport: { width: 1440, height: 900 } });
await c.goto(base + "/");
const add = c.getByRole("button", { name: /^Добавить в корзину: /, disabled: false }).first();
const addName = await add.getAttribute("aria-label");
await add.click();
out("after + : stepper shows 1", await c.locator("[aria-live=polite]").first().innerText());
out(
  "header cart badge",
  await c
    .getByRole("link", { name: /Корзина/ })
    .first()
    .getAttribute("aria-label"),
);
await c.getByRole("button", { name: "Добавить порцию" }).click();
out("after second + : qty", await c.locator("[aria-live=polite]").first().innerText());
out("localStorage", await c.evaluate(() => localStorage.getItem("citypark-cart")));
await c.reload();
await c.waitForTimeout(300);
out("after reload still in cart (qty)", await c.locator("[aria-live=polite]").first().innerText());
await c.getByRole("button", { name: "Убрать порцию" }).click();
await c.getByRole("button", { name: "Убрать порцию" }).click();
out(
  "removed to 0 -> '+' is back",
  String(await c.getByRole("button", { name: addName, exact: true }).count()),
);
out("localStorage cleared", String(await c.evaluate(() => localStorage.getItem("citypark-cart"))));

// ---- booking form
const f = await browser.newPage({ viewport: { width: 1440, height: 900 } });
await f.goto(base + "/");
await f.waitForTimeout(400);
await f.getByRole("button", { name: "Отправить заявку" }).click();
await f.waitForTimeout(600);
out(
  "empty phone -> error",
  await f
    .getByRole("alert")
    .first()
    .innerText()
    .catch(() => "NO ERROR SHOWN"),
);
const phone = f.getByLabel("Телефон");
await phone.fill("");
await phone.pressSequentially("89183119791");
out("phone mask", await phone.inputValue());
await phone.fill("");
await phone.pressSequentially("9183");
out("phone mask partial (9183)", await phone.inputValue());
await f.locator("form input[name=date]").fill("2020-01-01");
await phone.fill("");
await phone.pressSequentially("89183119791");
await f.getByRole("button", { name: "Отправить заявку" }).click();
await f.waitForTimeout(600);
out(
  "past date -> error",
  await f
    .getByRole("alert")
    .first()
    .innerText()
    .catch(() => "NO ERROR SHOWN"),
);
out("typed phone kept after error", await f.getByLabel("Телефон").inputValue());
await f.locator("form input[name=date]").fill(iso(now));
await f.getByRole("button", { name: "Отправить заявку" }).click();
await f.waitForTimeout(800);
out(
  "success message",
  await f
    .getByRole("status")
    .innerText()
    .catch(() => "NO SUCCESS"),
);

// ---- mobile: no overflow, buttons hidden
const m = await browser.newPage({ viewport: { width: 390, height: 844 } });
await m.goto(base + "/");
const vis = await m.evaluate(() => ({
  scrollW: document.documentElement.scrollWidth,
  viselect: [...document.querySelectorAll("a")].filter((a) => a.textContent.trim() === "Выбрать" && a.offsetParent)
    .length,
  email: [...document.querySelectorAll("a")].filter((a) => a.textContent.includes("belcitypark") && a.offsetParent)
    .length,
}));
out("mobile overflow / visible 'Выбрать' / visible e-mail", JSON.stringify(vis));
await browser.close();
