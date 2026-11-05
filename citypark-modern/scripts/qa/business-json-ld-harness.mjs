import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import { chromium } from 'playwright'

function load(file, modules) {
 const exports = {}
 const source = ts.transpileModule(readFileSync(file, 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
 }).outputText
 vm.runInNewContext(source, { exports, require: name => {
  assert.ok(name in modules, `unexpected dependency ${name}`)
  return modules[name]
 } })
 return exports
}
const { productionOrigin } = load('src/shared/config/search.ts', {})
const { site } = load('src/shared/config/site.ts', {})
const { defaultSiteSettings } = load('src/shared/config/site-settings.ts', {
 '@/shared/lib': { nb: text => text.replaceAll(' ', '\u00a0') },
})
const { businessJsonLd } = load('src/pages/contacts/model/business-json-ld.ts', {
 '@/shared/config': { productionOrigin, site },
})
const settings = structuredClone(defaultSiteSettings)
const hours = { start: '09:00', end: '22:00' }
const graph = JSON.parse(businessJsonLd(settings, hours))
assert.equal(graph['@context'], 'https://schema.org')
assert.equal(graph['@graph'].length, 2)
for (const [index, key, type, path] of [[0, 'hotel', 'Hotel', '/hotel'], [1, 'restaurant', 'Restaurant', '/restaurant']]) {
 const node = graph['@graph'][index]
 assert.equal(node['@type'], type)
 assert.equal(node['@id'], productionOrigin + path + '#business')
 assert.equal(node.url, productionOrigin + path)
 assert.equal(node.name, site.name)
 assert.equal(node.telephone, settings.phones[key].href.slice(4))
 assert.equal(node.email, settings.email)
 assert.deepEqual(node.address, {
  '@type': 'PostalAddress', streetAddress: settings.address.street,
  addressLocality: settings.address.city, postalCode: settings.address.postcode,
 })
 for (const omitted of ['aggregateRating', 'starRating', 'offers', 'availability', 'priceRange', 'geo', 'sameAs']) assert.ok(!(omitted in node))
}
assert.equal(graph['@graph'][1].openingHours, 'Mo-Su 09:00-22:00')
assert.ok(!('openingHours' in graph['@graph'][0]))

const malicious = '</script><img id="injected" src=x onerror="alert(1)"> <script> " & \\ \u2028\u2029 Привет'
settings.address = { street: malicious, city: malicious, postcode: malicious, full: malicious }
settings.email = malicious
settings.phones.hotel.href = 'tel:' + malicious
settings.phones.restaurant.href = 'tel:+79990000000'
const serialized = businessJsonLd(settings, { start: '22:00', end: '02:00' })
assert.ok(!serialized.includes('<'))
const changed = JSON.parse(serialized)['@graph']
assert.equal(changed[0].email, malicious)
assert.equal(changed[0].address.streetAddress, malicious)
assert.equal(changed[0].address.addressLocality, malicious)
assert.equal(changed[0].address.postalCode, malicious)
assert.equal(changed[0].telephone, malicious)
assert.equal(changed[1].telephone, '+79990000000')
assert.equal(changed[1].openingHours, 'Mo-Su 22:00-02:00')
assert.equal(changed[0]['@id'], graph['@graph'][0]['@id'])
settings.email = ''
settings.address.street = ''
assert.equal(JSON.parse(businessJsonLd(settings, hours))['@graph'][0].email, '')
assert.equal(JSON.parse(businessJsonLd(settings, hours))['@graph'][0].address.streetAddress, '')
const browser = await chromium.launch({ headless: true })
try {
 const page = await browser.newPage()
 const result = await page.evaluate(text => {
  const doc = new DOMParser().parseFromString(`<script type="application/ld+json">${text}</script>`, 'text/html')
  return { scripts: doc.querySelectorAll('script').length, injected: !!doc.querySelector('#injected'), json: JSON.parse(doc.querySelector('script').textContent) }
 }, serialized)
 assert.equal(result.scripts, 1)
 assert.equal(result.injected, false)
 assert.equal(result.json['@graph'][0].email, malicious)
} finally { await browser.close() }
console.log('PASS: graph fields, stable IDs, distinct phones, changed/empty settings, overnight hours and script-injection round trip')
