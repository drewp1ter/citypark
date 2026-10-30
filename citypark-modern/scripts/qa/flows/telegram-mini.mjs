import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/+$/, '')
const browser = await chromium.launch({ headless: true })

try {
	for (const viewport of [
		{ width: 390, height: 844 },
		{ width: 1440, height: 900 },
	]) {
		const context = await browser.newContext({ viewport })
		const page = await context.newPage()
		await page.route('https://telegram.org/js/telegram-web-app.js', (route) => route.abort())
		const response = await page.goto(`${base}/mini`)
		assert.equal(response.status(), 200)
		await page.getByText('Откройте приложение через Telegram').waitFor()
		assert.equal(await page.getByRole('heading', { name: 'Меню' }).count(), 0)
		assert.equal(await page.locator('form#checkout').count(), 0)
		assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
		await page.goto(`${base}/mini/checkout`)
		await page.getByText('Откройте приложение через Telegram').waitFor()
		assert.equal(await page.locator('form#checkout').count(), 0)
		console.log(`SDK-free ${viewport.width}px launch and checkout fail closed: PASS`)
		await context.close()
	}

	const context = await browser.newContext({ viewport: { width: 390, height: 844 } })
	const page = await context.newPage()
	await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/javascript',
			body: 'window.Telegram={WebApp:{initData:"",ready(){},expand(){}}}',
		})
	)
	await page.goto(`${base}/mini`)
	await page.getByText('Откройте приложение через Telegram').waitFor()
	assert.equal(await page.locator('form#checkout').count(), 0)
	console.log('SDK without raw initData fails closed: PASS')
	await context.close()

	const stale = await browser.newContext({ viewport: { width: 390, height: 844 } })
	await stale.addCookies([
		{
			name: '__Host-citypark_mini',
			value: 'cs_stale_qa_token',
			domain: new URL(base).hostname,
			path: '/',
			secure: true,
			httpOnly: true,
			sameSite: 'Lax',
		},
	])
	const stalePage = await stale.newPage()
	await stalePage.route('https://telegram.org/js/telegram-web-app.js', (route) => route.abort())
	await stalePage.goto(`${base}/mini/checkout`)
	await stalePage.getByText('Откройте приложение через Telegram').waitFor()
	assert.equal(await stalePage.locator('form#checkout').count(), 0)
	console.log('Stale mini cookie cannot open checkout as guest: PASS')
	await stale.close()

	// An owner-supplied backend session permits read-only checks of the simulated SDK UI.
	if (process.env.QA_MINI_SESSION) {
		const context = await browser.newContext({ viewport: { width: 390, height: 844 } })
		const origin = new URL(base)
		await context.addCookies([
			{
				name: '__Host-citypark_mini',
				value: process.env.QA_MINI_SESSION,
				domain: origin.hostname,
				path: '/',
				secure: true,
				httpOnly: true,
				sameSite: 'Lax',
			},
			{
				name: '__Host-citypark_customer',
				value: 'cs_unrelated_web_session',
				domain: origin.hostname,
				path: '/',
				secure: true,
				httpOnly: true,
				sameSite: 'Lax',
			},
		])
		const page = await context.newPage()
		await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/javascript',
				body: `window.__miniQa={events:{},shown:false,ready:false,expanded:false};window.Telegram={WebApp:{initData:'sdk-ui-only',viewportHeight:700,safeAreaInset:{top:12,bottom:18},contentSafeAreaInset:{bottom:9},ready(){__miniQa.ready=true},expand(){__miniQa.expanded=true},onEvent(n,f){(__miniQa.events[n]??=[]).push(f)},offEvent(n,f){__miniQa.events[n]=(__miniQa.events[n]??[]).filter(x=>x!==f)},BackButton:{show(){__miniQa.shown=true},hide(){__miniQa.shown=false},onClick(f){__miniQa.back=f},offClick(f){if(__miniQa.back===f)__miniQa.back=null}}}}`,
			})
		)
		await page.goto(`${base}/mini`)
		await page.getByRole('heading', { name: 'Меню' }).waitFor()
		assert.equal(
			(await context.cookies()).some((cookie) => cookie.name === '__Host-citypark_customer'),
			true
		)
		assert.equal(await page.evaluate(() => __miniQa.ready && __miniQa.expanded), true)
		assert.equal(await page.evaluate(() => __miniQa.shown), false)
		assert.equal(
			await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--mini-bottom').trim()),
			'27px'
		)
		await page.evaluate(() => {
			Telegram.WebApp.viewportHeight = 650
			__miniQa.events.viewportChanged.forEach((f) => f())
		})
		assert.equal(
			await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--mini-height').trim()),
			'650px'
		)
		await page.goto(`${base}/mini/checkout`)
		await page.getByRole('heading', { name: 'Оформление заказа' }).waitFor()
		assert.equal(await page.evaluate(() => __miniQa.shown), true)
		await page.evaluate(() => __miniQa.back())
		await page.waitForURL(`${base}/mini`)
		assert.equal(await page.evaluate(() => __miniQa.shown), false)
		assert.equal(await page.evaluate(() => Object.values(__miniQa.events).every((list) => list.length === 1)), true)
		assert.equal(await page.evaluate(() => __miniQa.back == null), true)
		console.log('Authenticated SDK UI, insets, BackButton and route listener cleanup: PASS')
		await context.close()
	} else {
		console.log('Authenticated SDK UI checks: skipped (QA_MINI_SESSION not supplied)')
	}

	// These real authentication POSTs need a fresh owner-approved Telegram launch.
	if (process.env.QA_CUSTOMER_WRITES === '1') {
		const raw = process.env.QA_TELEGRAM_INIT_DATA
		assert.ok(raw, 'QA_TELEGRAM_INIT_DATA is required for approved live checks')
		const context = await browser.newContext({ viewport: { width: 390, height: 844 } })
		const page = await context.newPage()
		const sdk = (value) => `window.Telegram={WebApp:{initData:${JSON.stringify(value)},ready(){},expand(){}}}`
		await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
			route.fulfill({ status: 200, contentType: 'application/javascript', body: sdk(raw) })
		)
		await page.goto(`${base}/mini`)
		await page.getByRole('heading', { name: 'Меню' }).waitFor()
		assert.equal(
			(await context.cookies()).some((cookie) => cookie.name === '__Host-citypark_mini'),
			true
		)
		await page.reload()
		await page.getByRole('heading', { name: 'Меню' }).waitFor()
		await page.goto(`${base}/mini/checkout`)
		await page.getByRole('heading', { name: 'Оформление заказа' }).waitFor()
		assert.equal(await page.locator('form#checkout input[name=phone]').count(), 1)
		console.log('Fresh launch, mini cookie reuse and checkout contact phone: PASS')
		await context.clearCookies()
		await page.goto(`${base}/mini`)
		await page.getByText('Не удалось войти. Закройте и снова откройте приложение').waitFor()
		console.log('Reload after lost cookie does not resubmit consumed proof: PASS')
		await context.close()

		for (const [label, proof] of [
			['invalid', 'invalid-proof'],
			['replayed', raw],
			['expired', process.env.QA_TELEGRAM_EXPIRED_INIT_DATA],
		].filter(([, value]) => value)) {
			const check = await browser.newContext()
			const checkPage = await check.newPage()
			await checkPage.route('https://telegram.org/js/telegram-web-app.js', (route) =>
				route.fulfill({ status: 200, contentType: 'application/javascript', body: sdk(proof) })
			)
			await checkPage.goto(`${base}/mini`)
			await checkPage.getByText('Не удалось войти. Закройте и снова откройте приложение').waitFor()
			console.log(`${label} launch fails closed: PASS`)
			await check.close()
		}
	} else {
		console.log('Live authentication/replay/order checks: skipped (owner approval and fresh Telegram launch required)')
	}
} finally {
	await browser.close()
}
