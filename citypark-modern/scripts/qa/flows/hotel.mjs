// Flow check: hotel. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/hotel.mjs   (server must run on BASE_URL, default http://localhost:3111)
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
async function open(path, viewport) {
  const ctx = await browser.newContext({ viewport });
  const p = await ctx.newPage();
  const problems = [];
  p.on("pageerror", (e) => problems.push(e.message.slice(0, 120)));
  const res = await p.goto(base + path);
  await p.waitForTimeout(500);
  return Object.assign(p, { problems, status: res.status() });
}
const D = { width: 1440, height: 900 },
  M = { width: 390, height: 844 };

let p = await open("/hotel", D);
out("desktop: status / h1 / title", `${p.status} / ${await txt(p.locator("h1"))} / ${await p.title()}`);
out("breadcrumbs", await txt(p.locator("nav[aria-label='Хлебные крошки']")));
out(
  "hero buttons -> hrefs",
  JSON.stringify(
    await p
      .locator("main section")
      .first()
      .locator("a:visible")
      .evaluateAll((a) => a.map((x) => `${x.textContent.trim().slice(0, 18)}→${x.getAttribute("href")}`).slice(2, 6)),
  ),
);
out(
  "room fund rows (order)",
  JSON.stringify(await p.locator("section:has(h2:text('Номерной фонд')) li strong").allInnerTexts()),
);
out("room fund: price row", await txt(p.locator("section:has(h2:text('Номерной фонд')) li").last()));
out("breakfast: 5 items on desktop", String(await p.locator("h3:text('Завтрак в номер') >> xpath=../ul/li").count()));
out(
  "rules: 6 cards, last is dark",
  `${await p.locator("section:has(h2:text-is('Правила')) h3").count()} / ${await p.locator("section:has(h2:text-is('Правила')) h3").last().innerText()}`,
);

// accordion (desktop info card)
const info = p.locator("section#info");
const items = info.locator("details");
out(
  "info: items / open at start",
  `${await items.count()} / ${await items.evaluateAll((d) => d.map((x) => (x.open ? 1 : 0)).join(""))}`,
);
await items.nth(1).locator("summary").click();
out(
  "click item 2 -> opens, shows 'Скоро добавим.'",
  `${await items.nth(1).evaluate((d) => d.open)} / ${await txt(items.nth(1).locator("div"))}`,
);
await items.nth(0).locator("summary").click();
out("click item 1 -> closes", String(await items.nth(0).evaluate((d) => d.open)));
await items.nth(2).locator("summary").focus();
await p.keyboard.press("Enter");
out("keyboard Enter toggles item 3", String(await items.nth(2).evaluate((d) => d.open)));
out(
  "documents: cards (not links yet)",
  `${await p.locator("#docs > div").count()} cards, ${await p.locator("#docs a").count()} links`,
);
out("header phone on /hotel", await p.getByRole("link", { name: "Позвонить в отель" }).first().getAttribute("href"));
out("nav: Отель is current", await txt(p.locator("nav[aria-label='Основное меню'] a[aria-current=page]")));
out(
  "desktop: overflow / errors",
  `${await p.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(p.problems)}`,
);

// mobile
const m = await open("/hotel", M);
out(
  "mobile: h1 / breadcrumbs hidden",
  `${await txt(m.locator("h1"))} / ${await m.locator("nav[aria-label='Хлебные крошки']").isVisible()}`,
);
out(
  "mobile: stats visible",
  JSON.stringify(
    await m
      .locator("ul li")
      .evaluateAll((l) =>
        l.filter((x) => x.offsetParent && /24\/7|0 ₽/.test(x.textContent)).map((x) => x.innerText.replace(/\s+/g, " ")),
      ),
  ),
);
out(
  "mobile: fund subtitle",
  await txt(m.locator("section:has(h2:text('Номерной фонд')) li").first().locator("span.text-\\[13px\\]")),
);
out(
  "mobile: breakfast items",
  await m.locator("h3:text('Завтрак в номер') >> xpath=../ul/li").evaluateAll((l) =>
    l
      .filter((x) => x.offsetParent)
      .map((x) => x.innerText.replace(/\s+/g, " "))
      .join(" | "),
  ),
);
out("mobile: photo/balconies card hidden", String(await m.getByText("Балконы над парком").isVisible()));
out("mobile: full info card hidden", String(await m.locator("section#info").isVisible()));
const top = m.locator("div.md\\:hidden > details");
out(
  "mobile: accordion rows / open",
  `${await top.count()} / ${await top.evaluateAll((d) => d.map((x) => (x.open ? 1 : 0)).join(""))}`,
);
out("mobile: row titles", (await top.locator("> summary").allInnerTexts()).join(" | "));
await top.nth(4).locator("> summary").click();
out(
  "mobile: Документы (PDF) opens, lists 3",
  `${await top.nth(4).evaluate((d) => d.open)} / ${await top.nth(4).locator("li").count()}`,
);
await top.nth(3).locator("> summary").click();
out(
  "mobile: Информация для гостей opens (nested items)",
  `${await top.nth(3).evaluate((d) => d.open)} / ${await top.nth(3).locator("details").count()} nested`,
);
out(
  "mobile: quick bar",
  `${await txt(m.locator("nav[aria-label='Быстрые действия']"))} ${JSON.stringify(await m.locator("nav[aria-label='Быстрые действия'] a").evaluateAll((a) => a.map((x) => x.getAttribute("href"))))}`,
);
out(
  "mobile: overflow / errors",
  `${await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(m.problems)}`,
);

// a11y
for (const [name, v] of [
  ["mobile", M],
  ["desktop", D],
]) {
  const q = await open("/hotel", v);
  const info2 = await q.evaluate(() => {
    const vis = (e) => e.offsetParent !== null || getComputedStyle(e).position === "fixed";
    const size = (e) => {
      const r = e.getBoundingClientRect();
      return [Math.round(r.width), Math.round(r.height)];
    };
    return {
      h1: document.querySelectorAll("h1").length,
      headings: [...document.querySelectorAll("h1,h2,h3")]
        .filter(vis)
        .map((h) => h.tagName)
        .join(""),
      imgNoAlt: [...document.images].filter((i) => !i.hasAttribute("alt")).length,
      small: [...document.querySelectorAll("a,button,summary")]
        .filter((e) => vis(e) && !e.classList.contains("tap-area") && size(e)[1] < 43.5)
        .map((e) => `${(e.getAttribute("aria-label") || e.textContent).trim().slice(0, 22)} ${size(e).join("x")}`),
    };
  });
  console.log(name.padEnd(8), JSON.stringify(info2));
}
await browser.close();
