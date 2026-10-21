// Full-page screenshots of routes at 390 and 1440 px in light and dark, for comparing with design/screenshots/.
// Usage: node scripts/screenshots.mjs <outDir> [route ...]   (server must run on BASE_URL, default http://localhost:3111)
// SEED_CART=1 puts a cart (1 + 2 + 1 portions of the first backend products) into localStorage first.
import { mkdirSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";
import { seedCart } from "./qa/menu-cart.mjs";

const [outDir, ...routes] = process.argv.slice(2);
if (!outDir) throw new Error("Usage: node scripts/screenshots.mjs <outDir> [route ...]");

const base = process.env.BASE_URL ?? "http://localhost:3111";
const sizes = { mobile: { width: 390, height: 844 }, desktop: { width: 1440, height: 900 } };
const themes = ["light", "dark"];

const browser = await chromium.launch();
for (const route of routes.length ? routes : ["/"]) {
  const name = route === "/" ? "home" : route.replace(/^\//, "").replaceAll("/", "-");
  for (const [sizeName, viewport] of Object.entries(sizes)) {
    for (const theme of themes) {
      const context = await browser.newContext({ viewport, deviceScaleFactor: 1, colorScheme: theme });
      await context.addInitScript((t) => localStorage.setItem("theme", t), theme);
      if (process.env.SEED_CART) {
        const cart = await seedCart([1, 2, 1]);
        await context.addInitScript((c) => localStorage.setItem("citypark-cart", JSON.stringify(c)), cart);
      }
      const page = await context.newPage();
      await page.goto(base + route, { waitUntil: "load" });
      await page.evaluate(() => document.fonts.ready);
      await page.waitForTimeout(300);
      // Sticky bars (mobile quick actions) should sit at the bottom of the page, as in design/screenshots.
      const height = await page.evaluate(() => document.documentElement.scrollHeight);
      await page.setViewportSize({ width: viewport.width, height });
      await page.waitForTimeout(150);
      const dir = join(outDir, theme, sizeName);
      mkdirSync(dir, { recursive: true });
      await page.screenshot({ path: join(dir, `${name}.png`), fullPage: true });
      await context.close();
    }
  }
}
await browser.close();
