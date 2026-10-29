// Flow check: PWA (manifest, icons, service worker, offline page). Prints what happened next to what is expected.
// Usage: node scripts/qa/flows/pwa.mjs   (server must run on BASE_URL, default http://localhost:3111)
import { chromium } from 'playwright'
const base = process.env.BASE_URL ?? 'http://localhost:3111'
const browser = await chromium.launch()
const out = (k, v) => console.log(k.padEnd(52), v)

// ---- manifest
const res = await fetch(base + '/manifest.webmanifest')
const manifest = await res.json()
out('manifest status', `${res.status}  (want 200)`)
out('manifest short_name', `${manifest.short_name}  (want Сити Парк)`)
out('manifest display', `${manifest.display}  (want standalone)`)
for (const size of ['192x192', '512x512']) {
	const icon = manifest.icons.find((i) => i.sizes === size && i.purpose === 'any')
	const r = icon ? await fetch(new URL(icon.src, base)) : undefined
	out(`icon ${size}`, `${r?.status} ${r?.headers.get('content-type')}  (want 200 image/png)`)
}

// ---- head tags
const html = await (await fetch(base + '/')).text()
out('<link rel="manifest">', `${/<link rel="manifest"/.test(html)}  (want true)`)
out('<link rel="apple-touch-icon">', `${/<link rel="apple-touch-icon"/.test(html)}  (want true)`)
out('<meta name="theme-color"> count', `${(html.match(/<meta name="theme-color"/g) ?? []).length}  (want 2)`)

// ---- sw.js headers
const sw = await fetch(base + '/sw.js')
out('sw.js Cache-Control', `${sw.headers.get('cache-control')}  (want to contain no-cache)`)

// ---- offline behaviour
const context = await browser.newContext()
const page = await context.newPage()
const fromWorker = []
page.on('response', (r) => {
	const u = r.url()
	if (r.fromServiceWorker() && (u.includes('/_next/') || /\.(png|jpe?g|webp|avif|svg)(\?|$)/.test(u))) fromWorker.push(u)
})
await page.goto(base + '/')
await page.evaluate(() => navigator.serviceWorker.ready)
await page.waitForTimeout(500)
await context.setOffline(true)
await page.goto(base + '/menu').catch(() => {})
out('offline /menu h1', `${await page.locator('h1').innerText()}  (want Нет соединения с интернетом)`)
await context.setOffline(false)
await page.goto(base + '/menu')
await page.waitForTimeout(500)
out('online /menu h1 is not the offline one', `${await page.locator('h1').first().innerText()}`)
out('assets served by the worker', `${fromWorker.length}  (want 0)`)

await browser.close()
