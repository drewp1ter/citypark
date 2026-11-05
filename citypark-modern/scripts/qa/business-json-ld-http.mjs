// Read-only production acceptance; parse initial HTML without executing website scripts.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
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
const browser = await chromium.launch({ headless: true })
try {
 const page = await browser.newPage()
 const result = await page.evaluate(source => {
  const doc = new DOMParser().parseFromString(source, 'text/html')
  const scripts = [...doc.querySelectorAll('script[type="application/ld+json"]')]
  const graphs = scripts.map(node => JSON.parse(node.textContent))
  const ids = scripts.map(node => node.id)
  doc.querySelectorAll('script,style,template').forEach(node => node.remove())
  return { graphs, ids, logos: [...doc.querySelectorAll('[role="img"][aria-label]')].map(node => node.getAttribute('aria-label')), text: doc.body.textContent, links: [...doc.querySelectorAll('a[href]')].map(node => node.getAttribute('href')) }
 }, html)
 assert.deepEqual(result.ids, ['business-structured-data'])
 const graph = result.graphs[0]
 assert.equal(graph['@context'], 'https://schema.org')
 assert.equal(graph['@graph'].length, 2)
 for (const [index, key, type, path] of [[0, 'hotel', 'Hotel', '/hotel'], [1, 'restaurant', 'Restaurant', '/restaurant']]) {
  const node = graph['@graph'][index]
  assert.equal(node['@type'], type)
  assert.equal(node['@id'], `https://cityparkvip.ru${path}#business`)
  assert.equal(node.url, `https://cityparkvip.ru${path}`)
  assert.equal(node.telephone, settings.phones[key].number)
  assert.equal(node.email, settings.email)
  assert.deepEqual(node.address, { '@type': 'PostalAddress', streetAddress: settings.address.street, addressLocality: settings.address.city, postalCode: settings.address.postcode })
  assert.ok(result.logos.includes(node.name), 'business name matches the visible logo accessible label')
  for (const fact of [node.email, ...Object.values(settings.address), settings.phones[key].display]) {
   assert.ok(normalize(result.text).includes(normalize(fact)), `visible fact: ${fact}`)
  }
  assert.ok(result.links.includes('tel:' + node.telephone))
  assert.ok(result.links.includes('mailto:' + node.email))
  assert.ok(result.links.includes(path), 'canonical business page is linked')
 }
 assert.equal(graph['@graph'][1].openingHours, `Mo-Su ${hours.start}-${hours.end}`)
 assert.ok(normalize(result.text).includes(`${hours.start.replace(/^0/, '')}–${hours.end.replace(/^0/, '')}`))
 console.log('PASS: one initial-HTML JSON-LD graph, Hotel/Restaurant IDs/URLs, live contacts/address/phones/hours and visible-content agreement')
} finally { await browser.close() }
