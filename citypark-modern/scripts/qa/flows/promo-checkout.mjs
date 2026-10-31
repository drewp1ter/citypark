// Production-build QA. Backend requests require owner approval under AGENTS.md.
// QA_PROMO_WRITES=1 creates temporary campaigns and disables them in finally; never creates orders/payments.
// QA_PROMO_SETTINGS=1 temporarily enables the global switch and restores settings in finally.
// QA_PROMO_CONFLICT=1 additionally permits a deliberately mismatched order POST, rejected with 409 before writes.
// Otherwise supply QA_PROMO_FIXTURES as JSON: {apply,replace,cap,overlap,expired,minimum,fee} campaign codes.
// Optional QA_MINI_SESSION / QA_CUSTOMER_SESSION exercise existing owner-supplied sessions; never sign in providers.
import assert from 'node:assert/strict'
import { chromium, expect } from 'playwright/test'

const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/+$/, '')
const api = process.env.MENU_API_URL?.replace(/\/+$/, '')
if (!api) throw new Error('Set MENU_API_URL to the configured existing backend')
const created = []
let token
let originalSettings
async function backend(path, body, method = 'POST') {
	const response = await fetch(api + path, {
		method: body === undefined ? 'GET' : method,
		headers: {
			'Content-Type': 'application/json',
			...(token && (path.startsWith('/promo-codes') || path === '/settings') ? { Authorization: `Bearer ${token}` } : {}),
		},
		...(body === undefined ? {} : { body: JSON.stringify(body) }),
	})
	const value = await response.json()
	assert(response.ok, `${path}: ${response.status} ${JSON.stringify(value)}`)
	return value
}
const townOption = (name) => new RegExp('^' + name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ' \\(')
const browser = await chromium.launch()
try {
	const categories = await backend('/categories')
	const visible = new Set(categories.map((category) => category.id))
	const products = (await backend('/products?pageSize=500')).filter(
		(product) => visible.has(product.categoryId) && product.isAvailableForOrder && product.price > 0
	)
	const towns = await backend('/delivery-prices')
	const town = towns.find((town) => town.price > 0 && town.freeFrom > 0)
	const otherTown = towns.find((item) => item.name !== town?.name && item.price > 0)
	assert(town && products.length > 1, 'Need two visible available dishes and a town with a free-delivery threshold')
	const product = products[0]
	const quantity = Math.ceil(town.freeFrom / product.price)
	const cart = [{ dishId: String(product.id), variantId: 'default', quantity }]
	let codes = JSON.parse(process.env.QA_PROMO_FIXTURES ?? '{}')
	if (process.env.QA_PROMO_WRITES === '1') {
		token = (
			await backend('/auth/sign-in', {
				email: process.env.QA_ADMIN_EMAIL ?? 'dev@dev.io',
				password: process.env.QA_ADMIN_PASSWORD ?? 'devdevdev',
			})
		).token
		const campaign = {
			percentage: 10,
			enabled: true,
			target: 'all',
			productIds: [],
			categoryIds: [],
			minimumEligibleSubtotal: null,
			maximumDiscount: null,
			startsAt: null,
			endsAt: null,
		}
		const variants = {
			apply: {},
			replace: { percentage: 20 },
			cap: { percentage: 99, maximumDiscount: 0.01 },
			overlap: { percentage: 50, target: 'selected', productIds: [product.id], categoryIds: [product.categoryId] },
			expired: { endsAt: '2020-01-01T00:00:00+03:00' },
			minimum: { minimumEligibleSubtotal: 100000000 },
			fee: { percentage: 99 },
		}
		for (const [name, options] of Object.entries(variants)) {
			const saved = await backend('/promo-codes', {
				...campaign,
				...options,
				code: `QAC_${Date.now()}_${name.toUpperCase()}`,
			})
			created.push(saved)
			codes[name] = saved.code
		}
	}
	if (process.env.QA_PROMO_SETTINGS === '1') {
		assert(token, 'QA_PROMO_SETTINGS requires QA_PROMO_WRITES=1')
		originalSettings = await backend('/settings')
		if (originalSettings.promoCodesEnabled) await backend('/settings', { ...originalSettings, promoCodesEnabled: false }, 'PUT')
		const disabledContext = await browser.newContext()
		await disabledContext.addInitScript(({cart}) => {
			localStorage.setItem('citypark-cart', JSON.stringify(cart))
			sessionStorage.setItem('citypark-checkout-promo:web', 'SAVED')
		}, {cart})
		const disabledPage = await disabledContext.newPage()
		await disabledPage.goto(base + '/checkout')
		await expect(disabledPage.getByRole('textbox', { name: /^Промокод/ })).toHaveCount(0)
		await expect.poll(() => disabledPage.evaluate(() => sessionStorage.getItem('citypark-checkout-promo:web'))).toBeNull()
		await expect(disabledPage.locator('input[name=promoCode]')).toHaveValue('')
		await expect(disabledPage.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })).toBeEnabled()
		await disabledContext.close()
		await backend('/settings', { ...originalSettings, promoCodesEnabled: true }, 'PUT')
		// Settings have a 60-second server cache. Poll fresh page loads until revalidation reaches checkout.
		const waitContext = await browser.newContext()
		await waitContext.addInitScript(({cart}) => localStorage.setItem('citypark-cart', JSON.stringify(cart)), {cart})
		const waitPage = await waitContext.newPage()
		await expect.poll(async () => {
			await waitPage.goto(base + '/checkout')
			return await waitPage.getByRole('textbox', { name: /^Промокод/ }).count()
		}, { timeout: 90000, intervals: [2000] }).toBe(1)
		await waitContext.close()
	}
	for (const name of ['apply', 'replace', 'cap', 'overlap', 'expired', 'minimum', 'fee'])
		assert(codes[name], `Missing QA_PROMO_FIXTURES.${name}; request approval before QA_PROMO_WRITES=1`)
	const money = (value) => `${new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 }).format(value)}\u00a0₽`
	const quote = (
		code,
		fulfillment = 'delivery',
		items = [{ productId: product.id, amount: quantity }],
		city = town.name
	) =>
		backend('/orders/quote', { items, fulfillment, ...(fulfillment === 'delivery' ? { city } : {}), promoCode: code })
	async function checkTotals(page, expected) {
		const values = [
			expected.subtotal,
			...(expected.promoCode ? [0 - expected.discount] : []),
			expected.deliveryPrice,
			expected.payableTotal,
		].map(money)
		await expect(page.locator('dl[aria-label="Расчёт заказа"] dd')).toHaveText(values)
		await expect(page.locator('input[name="expectedPayableTotal"]')).toHaveValue(String(expected.payableTotal))
		await expect(page.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })).toBeEnabled()
		console.log(
			`Quote payable: expected ${expected.payableTotal}, actual ${await page.locator('input[name=expectedPayableTotal]').inputValue()}`
		)
	}
	async function apply(page, code) {
		await page.getByRole('textbox', { name: /^Промокод/ }).fill(code)
		await expect(page.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })).toBeDisabled()

	}
	const surfaces = [{ path: '/checkout' }]
	if (process.env.QA_CUSTOMER_SESSION)
		surfaces.push({ path: '/checkout', session: process.env.QA_CUSTOMER_SESSION, cookie: '__Host-citypark_customer' })
	if (process.env.QA_MINI_SESSION)
		surfaces.push({ path: '/mini/checkout', session: process.env.QA_MINI_SESSION, cookie: '__Host-citypark_mini' })
	for (const surface of surfaces)
		for (const width of process.env.QA_CONFLICT_ONLY === '1' ? [390] : [390, 1440])
			for (const theme of process.env.QA_CONFLICT_ONLY === '1' ? ['light'] : ['light', 'dark']) {
				const context = await browser.newContext({ viewport: { width, height: 950 }, colorScheme: theme })
				await context.addInitScript(
					({ cart, theme }) => {
						localStorage.setItem('citypark-cart', JSON.stringify(cart))
						localStorage.setItem('theme', theme)
					},
					{ cart, theme }
				)
				if (surface.session)
					await context.addCookies([
						{ name: surface.cookie, value: surface.session, url: base, secure: true, httpOnly: true, sameSite: 'Lax' },
					])
				const page = await context.newPage()
				page.setDefaultTimeout(10000)
				if (surface.path.startsWith('/mini'))
					await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
						route.fulfill({
							contentType: 'application/javascript',
							body: `window.Telegram={WebApp:{initData:'qa-existing-session',ready(){},expand(){},onEvent(){},offEvent(){},BackButton:{show(){},hide(){},onClick(){},offClick(){}}}}`,
						})
					)
				let backendBrowserCalls = 0
				page.on('request', (request) => {
					if (request.url().startsWith(api)) backendBrowserCalls++
				})
				await page.goto(base + surface.path)
				{
					await page.getByRole('button', { name: /Населённый пункт/ }).click()
					await page.getByRole('option', { name: townOption(town.name) }).click()
				}
				await checkTotals(page, await quote(''))
				const promoField = page.getByRole('textbox', { name: /^Промокод/ })
				const clearPromo = page.getByRole('region', { name: 'Промокод', exact: true })
					.getByRole('button', { name: 'Очистить поле', exact: true })
				let actionCount = 0
				const countAction = (request) => {
					if (request.method() === 'POST' && request.headers()['next-action']) actionCount++
				}
				page.on('request', countAction)
				await promoField.fill('qa')
				await promoField.fill(`  ${codes.apply.toLowerCase()}  `)
				await expect(page.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })).toBeDisabled()
				await page.waitForTimeout(350)
				assert.equal(actionCount, 0, 'No quote may start before debounce settles')
				await checkTotals(page, await quote(codes.apply))
				assert.equal(actionCount, 1, 'Rapid typing must result in one quote')
				assert.equal(await page.locator('input[name=promoCode]').inputValue(), codes.apply)
				assert.equal(await page.evaluate((key) => sessionStorage.getItem(key), `citypark-checkout-promo:${surface.path.startsWith('/mini') ? 'mini' : 'web'}`), codes.apply)
				page.off('request', countAction)
				if (otherTown) {
					await promoField.fill(codes.replace)
					await page.getByRole('button', { name: /Населённый пункт/ }).click()
					await page.getByRole('option', { name: townOption(otherTown.name) }).click()
					await page.evaluate((cart) => {
						localStorage.setItem('citypark-cart', JSON.stringify(cart))
						window.dispatchEvent(new StorageEvent('storage', { key: 'citypark-cart' }))
					}, [{ ...cart[0], quantity: quantity + 1 }])
					await checkTotals(page, await quote(codes.replace, 'delivery', [{ productId: product.id, amount: quantity + 1 }], otherTown.name))
					await page.evaluate((cart) => {
						localStorage.setItem('citypark-cart', JSON.stringify(cart))
						window.dispatchEvent(new StorageEvent('storage', { key: 'citypark-cart' }))
					}, cart)
					await page.getByRole('button', { name: /Населённый пункт/ }).click()
					await page.getByRole('option', { name: townOption(town.name) }).click()
					await apply(page, codes.apply)
					await checkTotals(page, await quote(codes.apply))
				}
				await page.reload()
				await expect(promoField).toHaveValue(codes.apply)
				await checkTotals(page, await quote(codes.apply))
				if (process.env.QA_PROMO_SETTINGS === '1' && width === 390 && theme === 'light' && !surface.session) {
					await backend('/settings', { ...originalSettings, promoCodesEnabled: false }, 'PUT')
					await promoField.fill(codes.replace)
					await expect(page.getByRole('alert').filter({ hasText: 'Промокоды временно отключены.' })).toBeVisible()
					await expect(page.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })).toBeDisabled()
					await clearPromo.click()
					await checkTotals(page, await quote(''))
					await backend('/settings', { ...originalSettings, promoCodesEnabled: true }, 'PUT')
				}
				await promoField.fill('')
				await checkTotals(page, await quote(''))
				for (const name of ['apply', 'replace', 'cap', 'overlap', 'fee']) {
					await apply(page, codes[name])
					const expected = await quote(codes[name])
					if (name === 'cap') assert.equal(expected.discount, 0.01)
					if (name === 'overlap')
						assert.equal(expected.discount, Math.floor((expected.subtotal * 100 * 50) / 100) / 100)
					await checkTotals(page, expected)
				}
				await expect(page.getByText('После скидки доставка стала платной', { exact: true })).toBeVisible()
				if (otherTown) {
					await page.getByRole('button', { name: /Населённый пункт/ }).click()
					await page.getByRole('option', { name: townOption(otherTown.name) }).click()
					await checkTotals(page, await quote(codes.fee, 'delivery', undefined, otherTown.name))
					await page.getByRole('button', { name: /Населённый пункт/ }).click()
					await page.getByRole('option', { name: townOption(town.name) }).click()
					await checkTotals(page, await quote(codes.fee))
				}

				for (const [code, message] of [
					['!!', 'Промокод должен содержать'],
					[codes.expired, 'Промокод недействителен.'],
					[codes.minimum, 'Недостаточная сумма подходящих блюд для промокода.'],
				]) {
					await apply(page, code)
					await expect(page.getByRole('alert').filter({ hasText: message })).toBeVisible()
					await expect(page.getByRole('textbox', { name: /^Промокод/ })).toHaveValue(code)
					await expect(page.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })).toBeDisabled()
				}
				await clearPromo.click()
				await checkTotals(page, await quote(''))
				await apply(page, codes.apply)
				await checkTotals(page, await quote(codes.apply))
				await page.getByText('Самовывоз', { exact: true }).click()
				await checkTotals(page, await quote(codes.apply, 'pickup'))
				// Change the persisted cart in this tab through its normal storage subscription.
				const changed = [{ ...cart[0], quantity: quantity + 1 }]
				await page.evaluate((cart) => {
					localStorage.setItem('citypark-cart', JSON.stringify(cart))
					window.dispatchEvent(new StorageEvent('storage', { key: 'citypark-cart' }))
				}, changed)
				await checkTotals(page, await quote(codes.apply, 'pickup', [{ productId: product.id, amount: quantity + 1 }]))
				// Delay a real action response; removal must invalidate its result before it arrives.
				let held = false
				let release
				let intercepted
				const caught = new Promise((resolve) => {
					intercepted = resolve
				})
				const hold = new Promise((resolve) => {
					release = resolve
				})
				await page.route('**/*', async (route) => {
					if (held || route.request().method() !== 'POST' || !route.request().headers()['next-action'])
						return route.continue()
					held = true
					const response = await route.fetch()
					intercepted()
					await hold
					return route.fulfill({ response })
				})
				await apply(page, codes.replace)
				await caught
				await clearPromo.click()
				release()
				await checkTotals(page, await quote('', 'pickup', [{ productId: product.id, amount: quantity + 1 }]))
				await page.unrouteAll({ behavior: 'wait' })
				// A failed transport must keep the code and block submission until explicitly retried.
				await page.route('**/*', (route) =>
					route.request().method() === 'POST' && route.request().headers()['next-action']
						? route.abort('failed')
						: route.continue()
				)
				await apply(page, codes.cap)
				await expect(page.getByRole('alert').filter({ hasText: 'Не удалось рассчитать заказ' })).toBeVisible()
				await expect(page.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })).toBeDisabled()
				await page.unrouteAll({ behavior: 'wait' })
				await page.getByRole('textbox', { name: /^Промокод/ }).fill(codes.cap + 'X')
				await page.getByRole('textbox', { name: /^Промокод/ }).fill(codes.cap)
				await checkTotals(page, await quote(codes.cap, 'pickup', [{ productId: product.id, amount: quantity + 1 }]))
				if (process.env.QA_PROMO_CONFLICT === '1' && width === 390 && theme === 'light' && !surface.session) {
					// Explicitly authorized validation-only POST /orders: 0.01 cannot match this fixture's payable amount.
					// Never follow a provider redirect or resubmit after the conflict.
					const expected = await quote(codes.cap, 'pickup', [{ productId: product.id, amount: quantity + 1 }])
					assert(expected.payableTotal > 1)
					await page.getByLabel('Имя', { exact: true }).fill(`Тест конфликт ${Date.now()}`)
					await page.getByLabel('Телефон', { exact: true }).fill('')
					await page.getByLabel('Телефон', { exact: true }).pressSequentially('79999999999')
					await page.getByLabel(/Согласен/).check()
					await page.evaluate(() =>
						document
							.querySelector('form#checkout')
							.addEventListener('formdata', (event) => event.formData.set('expectedPayableTotal', '0.01'), {
								once: true,
							})
					)
					await page
						.getByRole('button', { name: /^Оплатить/ })
						.filter({ visible: true })
						.click()
					await expect(
						page
							.getByRole('alert')
							.filter({ hasText: 'Сумма заказа изменилась. Проверьте итог и повторите оформление.' })
					).toBeVisible()
					await checkTotals(page, expected)
					assert.equal(new URL(page.url()).pathname, surface.path)
					console.log('409 keeps the checkout open, fetches fresh pricing and waits for another submission: PASS')
				} else console.log('Validation-only 409 POST pending explicit QA_PROMO_CONFLICT=1 authorization')
				await page.screenshot({
					path: `/tmp/citypark-promo-${surface.path.startsWith('/mini') ? 'mini' : 'web'}-${width}-${theme}.png`,
					fullPage: true,
				})
				assert.equal(backendBrowserCalls, 0, 'Browsers must never call the backend')
				assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
				console.log(
					`${surface.path} ${width}px ${theme}: quotes, apply/replace/remove, errors, fee, cart, pickup, races and failed transport PASS`
				)
				await context.close()
			}
	if (!process.env.QA_MINI_SESSION)
		console.log(
			'Authenticated Mini App promo checks pending an owner-supplied QA_MINI_SESSION; telegram-mini regression checks the closed gate.'
		)
} finally {
	if (originalSettings) await backend('/settings', originalSettings, 'PUT')
	for (const campaign of created) await backend(`/promo-codes/${campaign.id}`, { ...campaign, enabled: false }, 'PUT')
	await browser.close()
}
