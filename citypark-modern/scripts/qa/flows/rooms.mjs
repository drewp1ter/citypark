// Flow check: rooms. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/rooms.mjs   (server must run on BASE_URL, default http://localhost:3111)
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
const iso = (d) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
const plus = (n) => {
  const d = new Date();
  d.setDate(d.getDate() + n);
  return iso(d);
};

async function open(path, viewport = { width: 1440, height: 900 }) {
  const ctx = await browser.newContext({ viewport });
  const p = await ctx.newPage();
  const problems = [];
  p.on("pageerror", (e) => problems.push(e.message.slice(0, 120)));
  const res = await p.goto(base + path);
  await p.waitForTimeout(500);
  return Object.assign(p, { problems, status: res.status() });
}

let p = await open("/rooms");
out(
  "default: status / h1 / cards",
  `${p.status} / ${await txt(p.locator("h1"))} / ${await p.locator("article").count()}`,
);
out("chip 'Все · 5 типов' (spacing)", await txt(p.locator("[role=group] a").first()));
out("VIP button (1 night default)", await txt(p.getByRole("link", { name: /Забронировать · / })));
out("statuses", JSON.stringify(await p.locator("article span.rounded-pill:visible").allInnerTexts()));
out(
  "search defaults today / tomorrow",
  `${await p.locator("input[name=checkIn]").inputValue()} / ${await p.locator("input[name=checkOut]").inputValue()}`,
);

// filters (links)
for (const [name, expected] of [
  ["Одноместные", 2],
  ["Двухместные", 2],
  ["С балконом", 2],
]) {
  await p.getByRole("link", { name, exact: true }).click();
  await p.waitForTimeout(400);
  out(
    `filter ${name}: url / cards`,
    `${p.url().replace(base, "")} / ${await p.locator("article").count()} (want ${expected})`,
  );
}
await p.getByRole("link", { name: "Только свободные" }).click();
await p.waitForTimeout(400);
out("+ only free (balcony)", `${p.url().replace(base, "")} / ${await p.locator("article h2").allInnerTexts()}`);
await p.getByRole("link", { name: "Двухместные", exact: true }).click();
await p.waitForTimeout(400);
out("double + free (keeps free)", `${p.url().replace(base, "")} / ${await p.locator("article h2").allInnerTexts()}`);

// search keeps working and filters survive
await p.goto(base + "/rooms?type=single");
await p.locator("input[name=checkIn]").fill(plus(3));
await p.waitForTimeout(100);
const out3 = await p.locator("input[name=checkOut]").inputValue();
out("check-out follows check-in +3", `${out3} (want ${plus(4)})`);
await p.getByRole("button", { name: "Найти" }).click();
await p.waitForURL("**/rooms?**");
await p.waitForTimeout(400);
out("search submit url", p.url().replace(base, ""));
out(
  "bar shows the searched dates",
  `${await p.locator("input[name=checkIn]").inputValue()} → ${await p.locator("input[name=checkOut]").inputValue()}, guests ${await p.locator("select[name=guests]").inputValue()}`,
);

// nights in the VIP price
const d = await open(`/rooms?checkIn=${plus(2)}&checkOut=${plus(4)}&guests=3-4`);
out("2 nights: VIP button (want 16 000 ₽)", await txt(d.getByRole("link", { name: /Забронировать · / })));
out("guests param -> select", await d.locator("select[name=guests]").inputValue());
// bad params are ignored
const bad = await open("/rooms?checkIn=2020-01-01&checkOut=nope&guests=99&type=hack&free=1");
out("bad params: status / cards (free only)", `${bad.status} / ${await bad.locator("article").count()}`);
out(
  "bad params: dates fall back",
  `${await bad.locator("input[name=checkIn]").inputValue()} / ${await bad.locator("input[name=checkOut]").inputValue()}`,
);
// nothing matches
const none = await open("/rooms?type=single&free=1");
await none.goto(base + "/rooms?type=double&free=1");
const empty = await open("/rooms?type=balcony&free=1");
out("balcony + free cards", String(await empty.locator("article").count()));
// make it empty: no room is both double and ... (euro is unavailable, twin is available) -> use an impossible mix via free+ type with none
// mobile
const m = await open("/rooms", { width: 390, height: 844 });
out(
  "mobile: cards / only-free chip hidden",
  `${await m.locator("article").count()} / ${await m.getByRole("link", { name: "Только свободные" }).isVisible()}`,
);
out("mobile: quick bar", await txt(m.locator("nav[aria-label='Быстрые действия']")));
out(
  "mobile: quick bar hrefs",
  JSON.stringify(
    await m.locator("nav[aria-label='Быстрые действия'] a").evaluateAll((as) => as.map((a) => a.getAttribute("href"))),
  ),
);
out("mobile: overflow", String(await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)));
out("mobile: amenities (short list)", await txt(m.locator("ul:visible").last()));
out("mobile: search button", await txt(m.getByRole("button", { name: /Найти/ })));
await m.getByRole("button", { name: /Найти/ }).click();
await m.waitForURL("**/rooms?**");
out("mobile submit keeps guests", m.url().replace(base, ""));
// hotel header phone + desktop phone link
out("header phone on /rooms", await p.getByRole("link", { name: "Позвонить в отель" }).first().getAttribute("href"));
// tablet
const t = await open("/rooms", { width: 900, height: 900 });
out(
  "tablet 900: grid cols / overflow",
  `${await t.evaluate(() => getComputedStyle(document.querySelector("article").parentElement).gridTemplateColumns.split(" ").length)} / ${await t.evaluate(() => document.documentElement.scrollWidth > innerWidth)}`,
);
out("page errors", JSON.stringify([...p.problems, ...m.problems, ...t.problems]));
await browser.close();
