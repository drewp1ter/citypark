// Safe by default. QA_CUSTOMER_WRITES=1 explicitly permits account writes on the configured backend.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { writeFileSync } from 'node:fs'
import { seedCart } from '../menu-cart.mjs'

const base = process.env.BASE_URL ?? 'http://localhost:3111'
const out = (label, value) => console.log(label.padEnd(46), value)
const browser = await chromium.launch()

for (const width of [390, 1440]) {
	for (const theme of ['light', 'dark']) {
		const context = await browser.newContext({ viewport: { width, height: 900 } })
		await context.addInitScript((value) => localStorage.setItem('theme', value), theme)
		const page = await context.newPage()
		const writes = []
		page.on('request', (request) => {
			if (request.method() !== 'GET' && request.url().includes('/auth/customer/')) writes.push(request.url())
		})
		const signIn = await page.goto(`${base}/account/sign-in?returnTo=%2F%2Fevil.example`)
		assert.equal(signIn.status(), 200)
		assert.equal(await page.locator('h1').innerText(), 'Вход')
		assert.match(signIn.headers()['cache-control'] ?? '', /private.*no-store/)
		assert.equal(signIn.headers()['referrer-policy'], 'no-referrer')
		out(`${width} ${theme} sign-in`, `${signIn.status()} ${await page.locator('h1').innerText()}`)
		out(`${width} ${theme} private headers`, `${signIn.headers()['cache-control']} | ${signIn.headers()['referrer-policy']}`)
		out(`${width} ${theme} horizontal overflow`, await page.evaluate(() => document.documentElement.scrollWidth > innerWidth))
		await page.getByLabel('Email').fill('invalid-email')
		await page.getByLabel('Пароль').fill('secret-password-for-test')
		await page.getByRole('button', { name: 'Войти' }).click()
		assert.equal(await page.getByLabel('Email').inputValue(), 'invalid-email')
		assert.ok(!(await page.locator('body').innerText()).includes('secret-password-for-test'))
		assert.equal(writes.length, 0)
		out(`${width} ${theme} invalid email stayed`, await page.getByLabel('Email').inputValue())
		out(`${width} ${theme} password not echoed`, !(await page.locator('body').innerText()).includes('secret-password-for-test'))
		const verify = await page.goto(`${base}/account/verify?token=qa-do-not-consume`)
		assert.equal(verify.status(), 200)
		assert.equal(writes.length, 0)
		assert.match(verify.headers()['cache-control'] ?? '', /private.*no-store/)
		assert.equal(verify.headers()['referrer-policy'], 'no-referrer')
		assert.equal(await page.getByRole('button', { name: 'Подтвердить email' }).count(), 1)
		out(`${width} ${theme} verify GET`, `${verify.status()} | POST count ${writes.length} (want 0)`)
		out(`${width} ${theme} verify headers`, `${verify.headers()['cache-control']} | ${verify.headers()['referrer-policy']}`)
		out(`${width} ${theme} browser storage`, await page.evaluate(() => ({ local: Object.keys(localStorage), session: Object.keys(sessionStorage) })))
		assert.deepEqual(await page.evaluate(() => Object.keys(sessionStorage)), [])
		assert.ok(!(await page.locator('body').innerText()).includes('cs_'))
		const register = await page.goto(`${base}/account/register`)
		assert.equal(register.status(), 200)
		assert.equal(await page.locator('h1').innerText(), 'Регистрация')
		assert.match(register.headers()['cache-control'] ?? '', /private.*no-store/)
		await context.close()
	}
}

const sw = await (await fetch(`${base}/sw.js`)).text()
out('service worker excludes account routes', sw.includes("pathname.startsWith('/account/')"))
assert.ok(sw.includes("pathname.startsWith('/account/')"))
if (process.env.QA_CUSTOMER_WRITES === '1') {
	const email = process.env.QA_CUSTOMER_EMAIL
	const password = process.env.QA_CUSTOMER_PASSWORD
	assert.ok(email && password, 'QA_CUSTOMER_EMAIL and QA_CUSTOMER_PASSWORD are required for live writes')
	const context = await browser.newContext()
	const page = await context.newPage()
	await page.goto(`${base}/account/register`)
	await page.getByLabel('Имя').fill('Тестовый клиент')
	await page.getByLabel('Email').first().fill(email)
	await page.getByLabel('Пароль').fill(password)
	await page.getByRole('button', { name: 'Зарегистрироваться' }).click()
	await assertGenericResult(page, 'registration')
	if (process.env.QA_CUSTOMER_VERIFY_TOKEN) {
		await page.goto(`${base}/account/verify?token=${encodeURIComponent(process.env.QA_CUSTOMER_VERIFY_TOKEN)}`)
		await page.getByRole('button', { name: 'Подтвердить email' }).click()
		await page.getByText('Email подтверждён. Теперь войдите в аккаунт.').waitFor()
		out('verification', 'confirmed by POST')
	} else {
		out('verification', 'skipped: provide QA_CUSTOMER_VERIFY_TOKEN from the latest email')
	}
	await page.goto(`${base}/account/register`)
	await page.getByLabel('Email').last().fill(email)
	await page.getByRole('button', { name: 'Отправить письмо повторно' }).click()
	await assertGenericResult(page, 'resend')
	await page.goto(`${base}/account/sign-in?returnTo=%2F%2Fevil.example`)
	await page.getByLabel('Email').fill(email)
	await page.getByLabel('Пароль').fill(password)
	await page.getByRole('button', { name: 'Войти' }).click()
	await page.waitForURL(`${base}/menu`)
	const session = (await context.cookies()).find((cookie) => cookie.name === '__Host-citypark_customer')
	assert.ok(session, 'sign-in did not set the customer cookie')
	assert.ok(session.httpOnly && session.secure && session.sameSite === 'Lax')
	assert.equal(session.path, '/')
	out('sign-in and unsafe return path', 'cookie flags valid; redirected to /menu')
	await page.goto(`${base}/account/sign-in`)
	await page.getByRole('button', { name: 'Выйти из аккаунта' }).click()
	await page.waitForURL(/\/account\/sign-in\?logout=/)
	assert.ok(!(await context.cookies()).some((cookie) => cookie.name === '__Host-citypark_customer'))
	out('logout', 'session cookie removed')
	await context.close()
} else {
	out('registration/resend/confirm/sign-in/logout', 'skipped: set QA_CUSTOMER_WRITES=1 with owner approval and a real test account')
}
if (process.env.QA_PLACE_ORDER === '1') {
	const { QA_CUSTOMER_EMAIL: email, QA_CUSTOMER_PASSWORD: password, QA_ORDER_NAME: name, QA_ORDER_PHONE: phone, QA_ORDER_COMMENT: comment, MENU_API_URL: api } = process.env
	assert.ok(email && password && name && phone && comment && api, 'QA_PLACE_ORDER needs customer credentials, order fields and MENU_API_URL')
	const cart = await seedCart([1])
	const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
	await context.addInitScript((lines) => localStorage.setItem('citypark-cart', JSON.stringify(lines)), cart)
	await context.addCookies([{ name: '__Host-citypark_customer', value: 'cs_expired_test_session', url: base.replace(/^http:/, 'https:'), secure: true, httpOnly: true, sameSite: 'Lax' }])
	const page = await context.newPage()
	await page.goto(`${base}/checkout`)
	await fillOrder(page, { name, phone, comment })
	await page.getByRole('button', { name: 'Оплатить заказ' }).click()
	await page.getByRole('alert').filter({ hasText: 'Войдите повторно, чтобы продолжить' }).waitFor()
	assert.equal(await page.getByLabel('Имя').inputValue(), name)
	assert.equal(await page.getByLabel('Пожелания').inputValue(), comment)
	assert.ok(await page.getByLabel(/Согласен/).isChecked(), 'consent was lost after the session error')
	assert.ok(await page.getByRole('radio', { name: 'Самовывоз' }).isChecked(), 'pickup selection was lost')
	assert.equal(JSON.parse(await page.evaluate(() => localStorage.getItem('citypark-cart'))).length, 1)
	out('expired session', '401 shown; name, comment and cart preserved')
	await context.clearCookies()
	await page.route('**/*', (route) => {
		if (route.request().isNavigationRequest() && new URL(route.request().url()).origin !== new URL(base).origin)
			return route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><title>Payment navigation blocked by QA</title>' })
		return route.continue()
	})
	const first = await submitForPayment(page)
	const guestCookie = (await context.cookies()).find((cookie) => cookie.name === 'citypark_guest_checkout_scope')
	assert.ok(guestCookie?.httpOnly && guestCookie.secure && guestCookie.sameSite === 'Lax' && guestCookie.path === '/')
	writeFileSync('/tmp/citypark-guest-order-qa-state.json', JSON.stringify(await context.storageState()), { mode: 0o600 })
	await page.waitForURL((url) => url.origin !== new URL(base).origin)
	await page.goto(`${base}/checkout`)
	await fillOrder(page, { name, phone, comment })
	const retry = await submitForPayment(page)
	assert.equal(retry, first, 'guest checkout retry created a second order')
	out('guest retry', `same order ${first}; scoped cookie flags valid`)
	await page.goto(`${base}/account/sign-in`)
	await page.getByLabel('Email').fill(email)
	await page.getByLabel('Пароль').fill(password)
	await page.getByRole('button', { name: 'Войти' }).click()
	await page.waitForURL(`${base}/menu`)
	const session = (await context.cookies()).find((cookie) => cookie.name === '__Host-citypark_customer')
	assert.ok(session)
	const history = await fetch(`${api.replace(/\/+$/, '')}/orders/my`, { headers: { Authorization: `Bearer ${session.value}` }, cache: 'no-store' })
	assert.equal(history.status, 200)
	const orders = await history.json()
	assert.ok(!orders.orders.some((order) => order.id === first), 'guest order appeared in account history')
	out('account history', `guest order ${first} excluded`)
	await context.close()
} else {
	out('guest and expired-session order retries', 'skipped: set QA_PLACE_ORDER=1 with owner approval')
}
await browser.close()

async function assertGenericResult(page, label) {
	const message = page.getByText('Если адрес доступен, письмо с подтверждением отправлено')
	await message.waitFor()
	out(label, await message.innerText())
}

async function fillOrder(page, { name, phone, comment }) {
	await page.getByText('Самовывоз', { exact: true }).click()
	await page.getByLabel('Имя').fill(name)
	await page.getByLabel('Телефон').fill('')
	await page.getByLabel('Телефон').pressSequentially(phone)
	await page.getByLabel('Пожелания').fill(comment)
	await page.getByLabel(/Согласен/).check()
}

async function submitForPayment(page) {
	const external = page.waitForEvent('request', {
		predicate: (request) => request.isNavigationRequest() && new URL(request.url()).origin !== new URL(base).origin,
		timeout: 8000,
	})
	await page.getByRole('button', { name: 'Оплатить заказ' }).click()
	let request
	try { request = await external }
	catch (error) {
		const alerts = await page.getByRole('alert').allInnerTexts()
		throw new Error(`No payment navigation; checkout alerts: ${JSON.stringify(alerts)}`, { cause: error })
	}
	const id = Number(new URL(request.url()).searchParams.get('MNT_TRANSACTION_ID'))
	assert.ok(Number.isInteger(id) && id > 0, 'payment URL did not contain a valid order ID')
	return id
}
