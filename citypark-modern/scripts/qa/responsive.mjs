// Responsive sweep: every page at many widths. Reports horizontal overflow (and the element that causes it),
// JavaScript errors and failed requests. Usage: node scripts/qa/responsive.mjs [route ...]
import { chromium } from "playwright";
import { seedCart } from "./menu-cart.mjs";

const base = process.env.BASE_URL ?? "http://localhost:3111";
const routes = process.argv.slice(2).length
  ? process.argv.slice(2)
  : [
      "/",
      "/restaurant",
      "/rooms",
      "/hotel",
      "/menu",
      "/checkout",
      "/checkout/done",
      "/news",
      "/news/frittata",
      "/contacts",
      "/nope",
    ];
const widths = [360, 390, 600, 768, 820, 1023, 1024, 1180, 1279, 1280, 1440, 1920];
const cart = await seedCart([2]);

const browser = await chromium.launch();
const problems = [];
for (const route of routes) {
  for (const width of widths) {
    const context = await browser.newContext({ viewport: { width, height: 900 } });
    await context.addInitScript((c) => localStorage.setItem("citypark-cart", JSON.stringify(c)), cart);
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (e) => errors.push(`pageerror: ${e.message.slice(0, 100)}`));
    page.on(
      "console",
      (m) => m.type() === "error" && !/404/.test(m.text()) && errors.push(`console: ${m.text().slice(0, 100)}`),
    );
    await page.goto(base + route, { waitUntil: "load" });
    await page.waitForTimeout(350);
    const overflow = await page.evaluate(() => {
      const root = document.documentElement;
      if (root.scrollWidth <= innerWidth) return null;
      const culprit = [...document.querySelectorAll("body *")]
        .filter((e) => e.getBoundingClientRect().right > innerWidth + 1 && getComputedStyle(e).position !== "fixed")
        .map(
          (e) =>
            `${e.tagName.toLowerCase()}.${String(e.className).split(" ").slice(0, 3).join(".")} (right=${Math.round(e.getBoundingClientRect().right)})`,
        )[0];
      return `scrollWidth ${root.scrollWidth} > ${innerWidth}; first culprit: ${culprit ?? "?"}`;
    });
    if (overflow) problems.push(`${route} @${width}: overflow ${overflow}`);
    for (const e of errors) problems.push(`${route} @${width}: ${e}`);
    await context.close();
  }
}
await browser.close();
console.log(`checked ${routes.length} routes x ${widths.length} widths`);
console.log(problems.length ? problems.join("\n") : "No overflow, no errors.");
process.exitCode = problems.length ? 1 : 0;
