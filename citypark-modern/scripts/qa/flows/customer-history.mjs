import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/+$/, '')
const browser = await chromium.launch()
try {
 for (const width of [390, 1440]) for (const theme of ['light', 'dark']) {
  const context = await browser.newContext({ viewport: { width, height: 900 } })
  await context.addInitScript(value => localStorage.setItem('theme', value), theme)
  const page = await context.newPage()
  for (const path of ['/account', '/account/orders', '/account/orders/1']) {
   const response = await page.goto(`${base}${path}`)
   await page.waitForURL(/\/account\/sign-in\?returnTo=/)
   assert.match(response.headers()['cache-control'] ?? '', /private.*no-store/)
   assert.equal(response.headers()['referrer-policy'], 'no-referrer')
   assert.equal(new URL(page.url()).searchParams.get('returnTo'), path)
  }
  await page.goto(`${base}/account/orders/not-an-id`)
  assert.equal(await page.getByText('This page could not be found.').count() > 0 || await page.locator('meta[name=robots][content=noindex]').count() > 0, true)
  assert.equal(await page.evaluate(() => Object.keys(sessionStorage).length), 0)
  assert.equal(await page.evaluate(() => Object.keys(localStorage).filter(key => key !== 'theme').length), 0)
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false)
  console.log(`${width} ${theme}: private redirects, cache headers, malformed ID, storage, layout PASS`)
  await context.close()
 }
 const sw = await (await fetch(`${base}/sw.js`)).text()
 assert.match(sw, /pathname === '\/account'/)
 assert.match(sw, /pathname\.startsWith\('\/account\/'\)/)
 console.log('Service worker excludes account and order history: PASS')
 if (process.env.QA_CUSTOMER_WRITES === '1') {
  const email = process.env.QA_CUSTOMER_EMAIL
  const password = process.env.QA_CUSTOMER_PASSWORD
  assert.ok(email && password, 'Provide QA_CUSTOMER_EMAIL and QA_CUSTOMER_PASSWORD for approved live scenarios')
  const context = await browser.newContext()
  const page = await context.newPage()
  await page.goto(`${base}/account/sign-in?returnTo=%2Faccount`)
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Пароль').fill(password)
  await page.getByRole('button', { name: 'Войти', exact: true }).click()
  await page.waitForURL(`${base}/account`)
  await page.getByRole('heading', { name: 'Личный кабинет' }).waitFor()
  await page.getByRole('heading', { name: 'Способы входа' }).waitFor()
  await page.getByRole('heading', { name: 'Подключить Telegram' }).waitFor()
  assert.equal(await page.getByRole('button', { name: 'Подтвердить связь' }).count(), 0)
  if (process.env.QA_TELEGRAM_PAIRING_CODE) {
   await page.getByLabel('Код из Telegram').fill(process.env.QA_TELEGRAM_PAIRING_CODE)
   await page.getByRole('button', { name: 'Продолжить' }).click()
   await page.getByRole('button', { name: 'Подтвердить связь' }).waitFor()
   assert.match(await page.locator('body').innerText(), /В Telegram:.*Здесь:/)
   console.log('Telegram redeem shows peer and both confirmation states without merging: PASS')
   if (process.env.QA_CONFIRM_PAIRING === '1') {
    await page.getByRole('button', { name: 'Подтвердить связь' }).click()
    await page.getByText(/Здесь: подтверждено|Аккаунты связаны/).waitFor()
    console.log('Website-side explicit pairing confirmation: PASS')
   } else console.log('Pairing confirmation: SKIPPED (set QA_CONFIRM_PAIRING=1 after approving account merge)')
  } else console.log('Telegram redeem and pairing states: SKIPPED (provide owner-approved QA_TELEGRAM_PAIRING_CODE)')
  await page.getByRole('link', { name: 'История заказов' }).click()
  await page.getByRole('heading', { name: 'История заказов' }).waitFor()
  const empty = await page.getByText('У вас пока нет заказов').count()
  if (!empty) {
   const links = page.getByRole('link', { name: /Заказ №/ })
   assert.ok(await links.count() <= 10)
   const href = await links.first().getAttribute('href')
   await links.first().click()
   await page.getByRole('heading', { name: /Заказ №/ }).waitFor()
   assert.equal(new URL(page.url()).pathname, href)
   if (process.env.QA_PROFILE_EDIT_NAME) {
    const recipientBefore = await page.getByText(/^Получатель:/).innerText()
    await page.goto(`${base}/account`)
    const originalName = await page.getByLabel('Имя').inputValue()
    await page.getByLabel('Имя').fill(process.env.QA_PROFILE_EDIT_NAME)
    await page.getByRole('button', { name: 'Сохранить' }).click()
    await page.getByText('Профиль сохранён').waitFor()
    await page.goto(`${base}${href}`)
    assert.equal(await page.getByText(/^Получатель:/).innerText(), recipientBefore)
    await page.goto(`${base}/account`)
    await page.getByLabel('Имя').fill(originalName)
    await page.getByRole('button', { name: 'Сохранить' }).click()
    await page.getByText('Профиль сохранён').waitFor()
    console.log('Order contact snapshot survives profile edit and profile restored: PASS')
   } else console.log('Snapshot-after-profile-edit: SKIPPED (provide QA_PROFILE_EDIT_NAME)')
  }
  console.log('Approved live account, methods, own history and detail: PASS')
  const secondEmail = process.env.QA_CUSTOMER_EMAIL_2
  const secondPassword = process.env.QA_CUSTOMER_PASSWORD_2
  const guestId = process.env.QA_GUEST_ORDER_ID
  if (secondEmail && secondPassword && guestId) {
   const api = process.env.MENU_API_URL
   assert.ok(api, 'MENU_API_URL is required for two-account ownership checks')
   async function signIn(address, secret) {
    const response = await fetch(`${api}/auth/customer/sign-in`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ email: address, password: secret }) })
    assert.equal(response.status, 200)
    return (await response.json()).token
   }
   async function read(token, path) {
    const response = await fetch(`${api}${path}`, { headers: { Authorization: `Bearer ${token}` }, cache: 'no-store' })
    assert.match(response.headers.get('cache-control') ?? '', /no-store/)
    return response
   }
   const firstToken = await signIn(email, password)
   const secondToken = await signIn(secondEmail, secondPassword)
   const firstPage = await read(firstToken, '/orders/my?page=1&pageSize=10')
   const secondPage = await read(secondToken, '/orders/my?page=1&pageSize=10')
   assert.equal(firstPage.status, 200)
   assert.equal(secondPage.status, 200)
   const firstList = await firstPage.json()
   const secondList = await secondPage.json()
   assert.ok(firstList.orders.length <= 10 && secondList.orders.length <= 10)
   assert.ok(firstList.count >= firstList.orders.length && secondList.count >= secondList.orders.length)
   assert.ok(firstList.orders.every(order => !secondList.orders.some(other => other.id === order.id)))
   if (process.env.QA_SHARED_PHONE) {
    assert.ok(firstList.orders.some(order => order.customer.phone === process.env.QA_SHARED_PHONE))
    assert.ok(secondList.orders.some(order => order.customer.phone === process.env.QA_SHARED_PHONE))
    console.log('Distinct owners with one contact phone: PASS')
   } else console.log('Shared contact phone: SKIPPED (provide QA_SHARED_PHONE fixture)')
   if (firstList.orders.length) {
    const id = firstList.orders[0].id
    assert.equal((await read(firstToken, `/orders/${id}`)).status, 200)
    assert.equal((await read(secondToken, `/orders/${id}`)).status, 404)
   }
   if (secondList.orders.length) assert.equal((await read(firstToken, `/orders/${secondList.orders[0].id}`)).status, 404)
   for (const token of [firstToken, secondToken]) assert.equal((await read(token, `/orders/${guestId}`)).status, 404)
   console.log('Two-account pagination, own detail, foreign 404 and guest exclusion: PASS')
  } else console.log('Two-account isolation and guest exclusion: SKIPPED (provide QA_CUSTOMER_EMAIL_2, QA_CUSTOMER_PASSWORD_2, QA_GUEST_ORDER_ID)')
  console.log('Real Telegram pairing needs a Mini App code and confirmation on both surfaces; SKIPPED')
  await context.close()
 } else {
  console.log('Live sign-in, profile writes, pairing, two-account ownership, guest exclusion and order scenarios: SKIPPED (QA_CUSTOMER_WRITES=1 and owner-approved fixtures required)')
 }
} finally { await browser.close() }
