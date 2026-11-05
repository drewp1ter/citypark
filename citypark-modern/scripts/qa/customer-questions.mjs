// Read-only public FAQ acceptance; no forms or backend data mutations.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
const require = createRequire(import.meta.url)
require('@next/env').loadEnvConfig(process.cwd())
const base = process.env.BASE_URL ?? 'http://localhost:3112'
const normalize = value => String(value).replace(/\s+/gu, ' ').trim()
async function api(path) {
 const response = await fetch(process.env.MENU_API_URL + path)
 assert.equal(response.status, 200, path)
 return response.json()
}
const [settings, hours] = await Promise.all([api('/settings'), api('/working-hours')])
const response = await fetch(base + '/contacts')
assert.equal(response.status, 200)
const html = await response.text()
const questions = [
 'Где находится «Сити Парк»?', 'Есть ли парковка?',
 'В какое время работают ресторан и доставка?',
 'Где посмотреть номера, цены и правила проживания?',
 'Где посмотреть меню доставки и цены на блюда?',
]
const expected = [settings.address.postcode, settings.address.city, settings.address.street,
 'Для гостей — бесплатная парковка у здания.',
 `${hours.start.replace(/^0/, '')}–${hours.end.replace(/^0/, '')}`,
 settings.phones.hotel.display, settings.phones.restaurant.display]
const links = ['/rooms', '/hotel', '/menu', `tel:${settings.phones.hotel.number}`, `tel:${settings.phones.restaurant.number}`]
const browser = await chromium.launch({ headless: true })
const artifacts = '/tmp/citypark-task65'
await mkdir(artifacts, { recursive: true })
try {
 const parser = await browser.newPage()
 const raw = await parser.evaluate(source => {
  const doc = new DOMParser().parseFromString(source, 'text/html')
  doc.querySelectorAll('script,style,template').forEach(node => node.remove())
  const section = doc.querySelector('#questions')
  return { text: section?.textContent ?? '', headings: [...(section?.querySelectorAll('h3') ?? [])].map(node => node.textContent), links: [...(section?.querySelectorAll('a') ?? [])].map(node => node.getAttribute('href')) }
 }, html)
 assert.deepEqual(raw.headings, questions)
 for (const text of expected) assert.ok(normalize(raw.text).includes(normalize(text)), `raw FAQ fact: ${text}`)
 for (const href of links) assert.ok(raw.links.includes(href), `raw FAQ link: ${href}`)
 await parser.close()
 for (const width of [390, 1440]) {
  for (const theme of ['light', 'dark']) {
   const context = await browser.newContext({ viewport: { width, height: 1000 }, colorScheme: theme, javaScriptEnabled: false })
   const page = await context.newPage()
   await page.goto(base + '/contacts', { waitUntil: 'load' })
   await page.evaluate(value => document.documentElement.setAttribute('data-theme', value), theme)
   const section = page.locator('#questions')
   await section.scrollIntoViewIfNeeded()
   assert.ok(await section.isVisible())
   assert.equal(await section.locator('h3').count(), questions.length)
   for (const heading of await section.locator('h3').all()) assert.ok(await heading.isVisible())
   for (const paragraph of await section.locator('p').all()) assert.ok(await paragraph.isVisible())
   assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `${width}/${theme}: overflow`)
   for (const phone of await section.locator('a[href^="tel:"]').all()) {
    assert.equal(await phone.evaluate(node => node.getClientRects().length), 1, 'telephone stays on one line')
   }
   const anchors = section.locator('a')
   await anchors.first().focus()
   assert.equal(await anchors.first().evaluate(node => node === document.activeElement), true)
   await page.keyboard.press('Tab')
   assert.equal(await anchors.nth(1).evaluate(node => node === document.activeElement), true)
   await section.screenshot({ path: `${artifacts}/questions-${theme}-${width}.png` })
   await context.close()
  }
 }
 console.log(`PASS: five raw-HTML answers, current contacts/hours and links; 390/1440px light/dark, keyboard, no overflow, no website JavaScript. Screenshots: ${artifacts}`)
} finally { await browser.close() }
