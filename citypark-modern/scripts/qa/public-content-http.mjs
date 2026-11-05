// Read-only acceptance: website scripts are never executed; serialized props cannot satisfy assertions.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { chromium } from 'playwright'
const require = createRequire(import.meta.url)
require('@next/env').loadEnvConfig(process.cwd())
const base = process.env.BASE_URL ?? 'http://localhost:3112'
const normalize = value => String(value).replace(/\s+/gu, ' ').trim()
const price = value => normalize(`${String(Math.round(value)).replace(/\B(?=(\d{3})+(?!\d))/g, ' ')} ₽`)
async function api(path) {
 const response = await fetch(process.env.MENU_API_URL + path)
 assert.equal(response.status, 200, `backend ${path}`)
 return response.json()
}
const [settings, hours, rooms, categories] = await Promise.all([
 api('/settings'), api('/working-hours'), api('/rooms'), api('/categories'),
])
assert.ok(rooms.length && categories.length, 'nonempty room and category coverage required')
const products = []
for (let page = 1; ; page++) {
 const batch = await api(`/products?pageSize=500&page=${page}`)
 products.push(...batch)
 if (batch.length < 500) break
}
assert.ok(products.length, 'nonempty product coverage required')
const browser = await chromium.launch({ headless: true })
try {
 const parser = await browser.newPage()
 async function parse(html) {
  return parser.evaluate(source => {
   const doc = new DOMParser().parseFromString(source, 'text/html')
   doc.querySelectorAll('script,style,template').forEach(node => node.remove())
   const main = doc.querySelector('main')
   return {
    text: main?.textContent ?? '',
    h1: doc.querySelectorAll('h1').length,
    headings: [...doc.querySelectorAll('h1')].map(node => node.textContent),
    links: [...doc.querySelectorAll('a[href]')].map(node => node.getAttribute('href')),
    articles: [...doc.querySelectorAll('main article')].map(node => ({ name: node.querySelector('h3')?.textContent, text: node.textContent })),
    body: doc.body.textContent,
   }
  }, html)
 }
 async function html(path) {
  const response = await fetch(base + path)
  assert.equal(response.status, 200, path)
  const result = await parse(await response.text())
  assert.equal(result.h1, 1, `${path}: one h1`)
  assert.ok(normalize(result.headings[0]), `${path}: nonempty heading`)
  assert.ok(normalize(result.text), `${path}: main content`)
  return result
 }
 function contains(text, expected, label) {
  assert.ok(normalize(text).includes(normalize(expected)), `${label}: missing ${normalize(expected)}`)
 }
 const isolated = await parse('<main>Visible fact<script>serialized-only-fact</script><template>template-only-fact</template><style>style-only-fact</style></main>')
 assert.equal(isolated.text, 'Visible fact', 'serialized and inert content must not pass as page text')
 const contacts = await html('/contacts')
 for (const fact of [...Object.values(settings.address), settings.email, ...Object.values(settings.company)]) contains(contacts.text, fact, 'contacts')
 for (const key of ['hotel', 'restaurant', 'free']) {
  assert.ok(contacts.links.includes(`tel:${settings.phones[key].number}`), `${key}: telephone link`)
  contains(contacts.text, settings.phones[key].display, `${key}: telephone text`)
 }
 const hoursText = `${hours.start.replace(/^0/, '')}–${hours.end.replace(/^0/, '')}`
 contains(contacts.text, hoursText, 'contact hours')
 contains((await html('/')).text, hoursText, 'home hours')
 await html('/restaurant')
 await html('/hotel')
 const listing = await html('/rooms')
 for (const room of rooms) {
  const path = `/rooms/${encodeURIComponent(room.id)}`
  assert.ok(listing.links.includes(path), `${path}: room link`)
  contains(listing.text, room.name, 'room listing name')
  contains(listing.text, price(room.pricePerNight), 'room listing price')
  if (room.summary) contains(listing.text, room.summary, 'room summary')
  const detail = await html(path)
  contains(detail.text, room.name, path)
  contains(detail.text, price(room.pricePerNight), path)
  if (room.description) {
   const description = (await parse(`<main>${room.description}</main>`)).text
   contains(detail.text, description, `${path}: description`)
  }
 }
 const menu = await html('/menu')
 contains(menu.text, hoursText, 'menu hours')
 let checkedProducts = 0
 for (const [index, category] of categories.entries()) {
  const path = index === 0 ? '/menu' : `/menu?category=${category.id}`
  assert.ok(menu.links.includes(path), `${path}: crawlable category`)
  const document = index === 0 ? menu : await html(path)
  contains(document.text, category.name, path)
  if (category.description.trim()) contains(document.text, category.description, `${path}: description`)
  for (const product of products.filter(item => item.categoryId === category.id)) {
   const card = document.articles.find(item => normalize(item.name) === normalize(product.name))
   assert.ok(card, `${path}: dish ${product.name}`)
   if (product.description.trim()) contains(card.text, product.description, product.name)
   contains(card.text, product.price > 0 ? price(product.price) : '—', `${product.name}: price`)
   checkedProducts++
  }
 }
 assert.ok(checkedProducts, 'products checked')
 console.log(`PASS: raw HTML contacts/hours, home/restaurant/hotel, ${rooms.length} room details, ${categories.length} category links, ${checkedProducts} dish descriptions/prices; no website JavaScript executed`)
} finally { await browser.close() }
