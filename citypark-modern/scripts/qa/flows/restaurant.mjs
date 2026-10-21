// Flow check: restaurant. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/restaurant.mjs   (server must run on BASE_URL, default http://localhost:3111)
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

let p = await open("/restaurant", D);
out("desktop: status / h1 / title", `${p.status} / ${await txt(p.locator("h1"))} / ${await p.title()}`);
out(
  "hero buttons",
  JSON.stringify(
    await p
      .locator("main section")
      .first()
      .locator("a:visible")
      .evaluateAll((a) => a.map((x) => `${x.textContent.trim().slice(0, 22)}→${x.getAttribute("href")}`).slice(2, 7)),
  ),
);
out(
  "facts (4)",
  JSON.stringify(
    await p
      .locator("main section")
      .first()
      .locator("ul li")
      .evaluateAll((l) => l.map((x) => x.innerText.replace(/\s+/g, " "))),
  ),
);
out(
  "sections: venues / banquets cards",
  `${await p.locator("#tour h2").allInnerTexts()} / ${(await p.locator("main section h2:text-is('Банкеты') >> xpath=../div/div/span[1]").allInnerTexts()).join(",")}`,
);
out(
  "header: nav current + phone",
  `${await txt(p.locator("nav[aria-label='Основное меню'] a[aria-current=page]"))} / ${await p.getByRole("link", { name: "Позвонить" }).first().getAttribute("href")}`,
);

// dark booking form
const form = p.locator("#booking form");
out(
  "form fields",
  JSON.stringify(await form.locator("label").evaluateAll((l) => l.map((x) => x.firstChild?.textContent.trim()))),
);
out("date default today", await form.locator("input[name=date]").inputValue());
await form.getByRole("button", { name: "Отправить заявку" }).click();
await p.waitForTimeout(700);
out(
  "empty submit -> alert + colour",
  `${await txt(form.getByRole("alert"))} / ${await form
    .getByRole("alert")
    .first()
    .evaluate((e) => getComputedStyle(e).color)}`,
);
await form.locator("input[name=date]").fill("2020-01-01");
await form.getByLabel("Телефон").pressSequentially("89183119791");
await form.getByRole("button", { name: "Отправить заявку" }).click();
await p.waitForTimeout(700);
out(
  "past date -> error, phone kept",
  `${(await form.getByRole("alert").allInnerTexts()).filter(Boolean)} / ${await form.getByLabel("Телефон").inputValue()}`,
);
await form.locator("input[name=date]").fill(iso(new Date()));
await form.getByLabel("Гостей").selectOption("5–8");
await form.getByLabel("Зал").selectOption("cafe");
await form.getByRole("button", { name: "Отправить заявку" }).click();
await p.waitForTimeout(900);
out("valid -> success", await txt(form.getByRole("status")));
out(
  "desktop: overflow / errors",
  `${await p.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(p.problems)}`,
);

// home form still works without the hall
const h = await open("/", D);
await h.getByLabel("Телефон").pressSequentially("89183119791");
await h.getByRole("button", { name: "Отправить заявку" }).click();
await h.waitForTimeout(900);
out("home booking form (no hall) -> success", await txt(h.getByRole("status")));

// mobile
const m = await open("/restaurant", M);
out(
  "mobile: breadcrumbs / h1",
  `${await txt(m.locator("nav[aria-label='Хлебные крошки']:visible"))} / ${await txt(m.locator("h1"))}`,
);
out(
  "mobile: photos in rail",
  String(await m.locator("main section").first().locator("div.overflow-x-auto img").count()),
);
out("mobile: venue cards (short texts)", (await m.locator("#tour p:visible").allInnerTexts()).join(" | "));
out("mobile: banquets", await txt(m.locator("main section").nth(2).locator("div:visible").first()));
out(
  "mobile: quick bar",
  `${await txt(m.locator("nav[aria-label='Быстрые действия']"))} ${JSON.stringify(await m.locator("nav[aria-label='Быстрые действия'] a").evaluateAll((a) => a.map((x) => x.getAttribute("href"))))}`,
);
await m.locator("nav[aria-label='Быстрые действия'] a").last().click();
await m.waitForTimeout(500);
out(
  "mobile: 'Столик' scrolls to the form",
  String(
    await m.locator("#booking").evaluate((e) => {
      const r = e.getBoundingClientRect();
      return r.top >= 0 && r.top < 200;
    }),
  ),
);
out(
  "mobile: left column hidden / heading shown",
  `${await m.locator("#booking div.hidden").first().isVisible()} / ${await txt(m.locator("#booking h2:visible"))}`,
);
out(
  "mobile: overflow / errors",
  `${await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(m.problems)}`,
);

for (const [name, v] of [
  ["mobile", M],
  ["desktop", D],
]) {
  const q = await open("/restaurant", v);
  const info = await q.evaluate(() => {
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
      unlabelled: [...document.querySelectorAll("input:not([type=hidden]),select")].filter(
        (i) => vis(i) && !i.labels?.length,
      ).length,
      small: [...document.querySelectorAll("a,button,input:not([type=hidden]),select")]
        .filter((e) => vis(e) && !e.classList.contains("tap-area") && size(e)[1] < 43.5)
        .map(
          (e) =>
            `${(e.getAttribute("aria-label") || e.textContent || e.name).trim().slice(0, 22)} ${size(e).join("x")}`,
        ),
    };
  });
  console.log(name.padEnd(8), JSON.stringify(info));
}
await browser.close();
