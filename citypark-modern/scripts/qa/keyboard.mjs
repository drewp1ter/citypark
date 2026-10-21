// Keyboard pass: skip link, a visible focus ring on every Tab stop, no lost focus, and the mobile menu focus trap.
// Usage: node scripts/qa/keyboard.mjs [route ...]
import { chromium } from "playwright";
import { seedCart } from "./menu-cart.mjs";

const base = process.env.BASE_URL ?? "http://localhost:3111";
const routes = process.argv.slice(2).length
  ? process.argv.slice(2)
  : ["/", "/restaurant", "/rooms", "/hotel", "/menu", "/checkout", "/news", "/news/frittata", "/contacts"];
const cart = await seedCart([1]);
const problems = [];
const report = [];

const browser = await chromium.launch();

async function describeFocus(page) {
  return page.evaluate(() => {
    const el = document.activeElement;
    if (!el || el === document.body) return null;
    const ringOf = (e) => {
      const s = getComputedStyle(e);
      return s.outlineStyle !== "none" && parseFloat(s.outlineWidth) > 0;
    };
    // Visually hidden radios/checkboxes show their ring on the wrapping label.
    const pickerStop =
      (el.type === "date" || el.type === "time") &&
      !el.matches(":focus-visible") &&
      [...document.styleSheets].some((sheet) => {
        try {
          return [...sheet.cssRules].some(
            (rule) =>
              rule.cssText.includes("calendar-picker-indicator:focus-visible") ||
              [...(rule.cssRules ?? [])].some((r) => r.cssText.includes("calendar-picker-indicator:focus-visible")),
          );
        } catch {
          return false;
        }
      });
    const ring = ringOf(el) || pickerStop || (el.closest("label") ? ringOf(el.closest("label")) : false);
    const r = el.getBoundingClientRect();
    return {
      name: (el.getAttribute("aria-label") || el.textContent || el.getAttribute("name") || el.tagName)
        .trim()
        .replace(/\s+/g, " ")
        .slice(0, 36),
      tag: el.tagName.toLowerCase(),
      ring,
      inView: r.bottom > 0 && r.top < innerHeight,
      hidden: getComputedStyle(el).visibility === "hidden" || (r.width === 0 && r.height === 0),
    };
  });
}

for (const [sizeName, viewport] of [
  ["desktop", { width: 1440, height: 900 }],
  ["mobile", { width: 390, height: 844 }],
]) {
  for (const route of routes) {
    const context = await browser.newContext({ viewport });
    await context.addInitScript((c) => localStorage.setItem("citypark-cart", JSON.stringify(c)), cart);
    const page = await context.newPage();
    await page.goto(base + route, { waitUntil: "load" });
    await page.waitForTimeout(400);

    // skip link
    await page.keyboard.press("Tab");
    const first = await describeFocus(page);
    if (!first || !/Перейти к содержимому/.test(first.name))
      problems.push(`${route} ${sizeName}: first Tab stop is not the skip link (${first?.name})`);
    else if (
      !first.inView ||
      (!first.ring && !(await page.evaluate(() => getComputedStyle(document.activeElement).position === "fixed")))
    )
      problems.push(`${route} ${sizeName}: skip link not visible when focused`);
    await page.keyboard.press("Enter");
    const target = await page.evaluate(() => document.activeElement?.id);
    if (target !== "content")
      problems.push(`${route} ${sizeName}: skip link did not move focus to #content (got "${target}")`);

    // tab through the whole page
    await page.goto(base + route, { waitUntil: "load" });
    await page.waitForTimeout(300);
    let stops = 0;
    const seen = new Set();
    for (let i = 0; i < 160; i++) {
      await page.keyboard.press("Tab");
      const f = await describeFocus(page);
      if (!f) break; // focus left the document (browser UI): the loop is complete
      const key = `${f.tag}:${f.name}:${i}`;
      stops++;
      if (!f.hidden && !f.ring) problems.push(`${route} ${sizeName}: no focus ring on ${f.tag} "${f.name}"`);
      if (f.hidden && f.tag !== "input") problems.push(`${route} ${sizeName}: focus on a hidden ${f.tag} "${f.name}"`);
      if (!f.inView) problems.push(`${route} ${sizeName}: focused ${f.tag} "${f.name}" is not scrolled into view`);
      seen.add(key);
    }
    report.push(`${route.padEnd(16)} ${sizeName.padEnd(8)} ${stops} tab stops`);
    await context.close();
  }
}

// mobile menu: native modal dialog = focus trap
{
  const context = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await context.newPage();
  await page.goto(base + "/", { waitUntil: "load" });
  await page.waitForTimeout(400);
  const opener = page.getByRole("button", { name: "Открыть меню" });
  await opener.focus();
  await page.keyboard.press("Enter");
  await page.waitForTimeout(200);
  // A modal dialog lets Tab leave to the browser's own UI (activeElement becomes body). The trap is broken only
  // if focus lands on an element of the page behind it.
  const inside = () =>
    page.evaluate(() => {
      const el = document.activeElement;
      return !el || el === document.body || !!el.closest("dialog[open]");
    });
  let allInside = await inside();
  for (let i = 0; i < 25; i++) {
    await page.keyboard.press("Tab");
    allInside &&= await inside();
  }
  report.push(`mobile menu: focus never reached the page behind the dialog in 25 Tab presses: ${allInside}`);
  if (!allInside) problems.push("mobile menu: focus escaped the dialog");
  await page.keyboard.press("Escape");
  await page.waitForTimeout(200);
  const closed = await page.evaluate(() => !document.querySelector("dialog[open]"));
  const back = await page.evaluate(() => document.activeElement?.getAttribute("aria-label"));
  report.push(`mobile menu: Esc closes: ${closed}; focus returns to: "${back}"`);
  if (!closed) problems.push("mobile menu: Esc did not close it");
  if (back !== "Открыть меню") problems.push(`mobile menu: focus did not return to the menu button (got "${back}")`);
  await context.close();
}

await browser.close();
console.log(report.join("\n"));
console.log(
  problems.length ? `\n${problems.length} problems:\n${[...new Set(problems)].join("\n")}` : "\nNo keyboard problems.",
);
process.exitCode = problems.length ? 1 : 0;
