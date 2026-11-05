// Production: BASE_URL=http://localhost:3111, MENU_API_URL from .env.
// Run isolated failures/crypto checks: node scripts/qa/payment-retry-harness.mjs.
// Optional read-only real integration: QA_WEB_SESSION / QA_MINI_SESSION, existing
// owned orders from GET /orders/my. No credentials or provider URLs are logged.
// Owner-run acceptance still required: invoke retry with an existing eligible
// unpaid discounted order, double-click/keyboard/pending checks, then complete
// provider success/failure on native Telegram and external browser. This flow
// does not POST orders, retry links, payments, or webhooks; never creates fixtures.
import assert from 'node:assert/strict'
import { createHmac } from 'node:crypto'
import { chromium } from 'playwright'
import { seedCart } from '../menu-cart.mjs'
const transactionId = '12345678-abcd-abcd-abcd-123456789abc'
const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/+$/, '')
const browser = await chromium.launch()
const cart = JSON.stringify(await seedCart([1]))
try {
 for (const width of [390, 1440]) for (const theme of ['light', 'dark']) {
  const context = await browser.newContext({ viewport: { width, height: 900 } })
  await context.addInitScript(({ cart, theme }) => { localStorage.setItem('citypark-cart', cart); localStorage.setItem('theme', theme) }, { cart, theme })
  const page = await context.newPage()
  await page.route('https://telegram.org/js/telegram-web-app.js', route => route.abort())
  for (const path of ['/account/orders', '/account/orders/1']) {
   await page.goto(base + path); assert.equal(new URL(page.url()).pathname, '/account/sign-in')
  }
  for (const path of [`/checkout/done?MNT_TRANSACTION_ID=${transactionId}&origin=history&payment=returned`, `/checkout?payment=failed&MNT_TRANSACTION_ID=${transactionId}&origin=history`]) {
   await page.goto(base + path)
   assert.equal(new URL(page.url()).pathname.startsWith('/account/orders'), false)
   assert.equal(await page.evaluate(() => localStorage.getItem('citypark-cart')), cart)
   assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
  }
  await page.reload(); assert.equal(await page.evaluate(() => localStorage.getItem('citypark-cart')), cart)
  console.log(`PASS expected safe missing context and unchanged cart: ${width}px ${theme}`)
  await context.close()
 }
 for (const surface of ['web', 'mini']) {
  const token = process.env[surface === 'web' ? 'QA_WEB_SESSION' : 'QA_MINI_SESSION']
  if (!token) { console.log(`PENDING ${surface}: owner-provided session required for real owned-order acceptance`); continue }
  const response = await fetch(`${process.env.MENU_API_URL}/orders/my?pageSize=10`, { headers: { Authorization: `Bearer ${token}` } })
  assert.equal(response.status, 200)
  const { orders } = await response.json(); assert.ok(orders.length, 'Existing owned QA orders required')
  for (const width of [390, 1440]) for (const theme of ['light', 'dark']) {
   const context = await browser.newContext({ viewport: { width, height: 900 } })
   await context.addInitScript(({ cart, theme }) => { localStorage.setItem('citypark-cart', cart); localStorage.setItem('theme', theme) }, { cart, theme })
   const cookie = { domain: new URL(base).hostname, path: '/', secure: true, httpOnly: true, sameSite: 'Lax' }
   await context.addCookies([{ ...cookie, name: surface === 'mini' ? '__Host-citypark_mini' : '__Host-citypark_customer', value: token }])
   const page = await context.newPage()
   await page.route('https://telegram.org/js/telegram-web-app.js', route => route.fulfill({ contentType: 'application/javascript', body: 'window.Telegram={WebApp:{initData:"qa",ready(){},expand(){}}}' }))
   const prefix = surface === 'mini' ? '/mini' : '/account'
   await page.goto(base + prefix + '/orders'); await page.getByRole('heading', { name: 'История заказов' }).waitFor()
   for (const order of orders.slice(0, 3)) {
    for (const failed of [false, true]) {
     const payload = Buffer.from(JSON.stringify({ orderId: order.id, transactionId, surface, origin: 'history', issuedAt: Date.now() })).toString('base64url')
     const signature = createHmac('sha256', token).update(payload).digest('base64url')
     await context.addCookies([{ ...cookie, name: '__Host-citypark_payment', value: `${payload}.${signature}` }])
     await page.goto(`${base}/checkout${failed ? '?payment=failed&' : '/done?'}MNT_TRANSACTION_ID=${transactionId}`)
     await page.waitForURL(url => url.pathname === `${prefix}/orders/${order.id}`)
     await page.getByRole('heading', { name: `Заказ № ${order.id}`, exact: true }).waitFor()
     assert.equal(await page.getByRole('button', { name: 'Оплатить', exact: true }).count(), order.paymentRetry?.allowed ? 1 : 0)
     if (['paid', 'confirmed'].includes(order.status)) await page.getByText('Заказ оплачен', { exact: true }).waitFor()
     await page.reload()
     assert.equal(await page.evaluate(() => localStorage.getItem('citypark-cart')), cart)
    }
   }
   console.log(`PASS real ${surface} owned history and signed repeated returns/cart: ${width}px ${theme}`)
   await context.close()
  }
 }
 console.log('PENDING: authorized retry POST, stale backend rejection, provider/device handoff and payment confirmation acceptance')
} finally { await browser.close() }
