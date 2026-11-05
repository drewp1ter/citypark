import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { chromium } from 'playwright'
const require = createRequire(import.meta.url)
require('@next/env').loadEnvConfig(process.cwd())
const base = process.env.BASE_URL ?? 'http://localhost:3112'
const response = await fetch(`${base}/sitemap.xml`)
assert.equal(response.status, 200)
assert.match(response.headers.get('content-type'), /xml/)
const xml = await response.text()
const browser = await chromium.launch({ headless: true })
try {
 const page = await browser.newPage()
 const urls = await page.evaluate(text => {
  const doc = new DOMParser().parseFromString(text, 'application/xml')
  if (doc.querySelector('parsererror')) throw Error('Invalid XML')
  if (doc.documentElement.namespaceURI !== 'http://www.sitemaps.org/schemas/sitemap/0.9') throw Error('Invalid namespace')
  return [...doc.getElementsByTagName('loc')].map(node => node.textContent)
 }, xml)
 const api = process.env.MENU_API_URL
 const roomsResponse = await fetch(`${api}/rooms`)
 assert.equal(roomsResponse.status, 200)
 const paths = ['/', '/restaurant', '/hotel', '/rooms', '/menu', '/news', '/contacts', '/documents', '/offer', '/privacy']
 for (const room of await roomsResponse.json()) paths.push(`/rooms/${encodeURIComponent(room.id)}`)
 for (let index = 1; ; index++) {
  const res = await fetch(`${api}/news?page=${index}&pageSize=100`)
  assert.equal(res.status, 200)
  const data = await res.json()
  for (const item of data.news) paths.push(`/news/${encodeURIComponent(item.slug)}`)
  if (!data.news.length || index * 100 >= data.count) break
 }
 assert.deepEqual(urls.slice().sort(), [...new Set(paths.map(path => `https://cityparkvip.ru${path}`))].sort())
 const robots = await fetch(`${base}/robots.txt`)
 assert.equal(robots.status, 200)
 assert.match(await robots.text(), /Sitemap: https:\/\/cityparkvip\.ru\/sitemap\.xml/)
 console.log(`PASS: valid sitemap XML, ${urls.length} exact public URLs from live backend, robots reference`)
} finally { await browser.close() }
