// Production build: npm run build && npm start -- -p 3111
// Guest checks need no backend writes. Backend at MENU_API_URL must already run.
// Account acceptance (owner authorization required): provide existing disposable customer
// sessions QA_WEB_SESSION, QA_OTHER_WEB_SESSION, QA_MINI_SESSION. Seed legacy/unavailable
// addresses on these accounts before running to verify their full raw text/correction.
// Authorized run: QA_ADDRESS_WRITES=1 QA_WEB_SESSION=... QA_OTHER_WEB_SESSION=...
// QA_MINI_SESSION=... node --env-file=.env scripts/qa/flows/my-addresses.mjs
// This flow creates/edits/defaults/deletes temporary addresses only. No orders or payments.
// Import consent/session checks use existing QA_WEB_SESSION / QA_MINI_SESSION (read-only
// backend profile/list requests plus disposable browser storage). Import writes additionally
// require QA_IMPORT_WRITES=1 and QA_ADDRESS_WRITES=1: explicitly accept/retry/dismiss,
// verify duplicates/defaults and remove only the temporary imported account rows in finally.
// Example owner run: QA_ADDRESS_WRITES=1 QA_IMPORT_WRITES=1 QA_WEB_SESSION=... \
// QA_MINI_SESSION=... node --env-file=.env scripts/qa/flows/my-addresses.mjs
// No login proofs, tokens or address text are logged. Password/Google login returning
// to /menu, /checkout and an arbitrary site page, plus native Mini App relaunch and
// keyboard acceptance, must be verified by the owner; supplied sessions do not prove them.
// Import acknowledgments remove only unchanged local revisions; committed account writes
// survive local cleanup/transport failures. Retrying safely reuses backend duplicates.
// Native Telegram relaunch/keyboard checks require an owner device and remain manual.
import assert from 'node:assert/strict'
import { chromium, expect } from 'playwright/test'
import { randomUUID } from 'node:crypto'
import { seedCart } from '../menu-cart.mjs'
const checkoutCart = await seedCart([1])
const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/+$/, '')
const browser = await chromium.launch()
const key = 'citypark.addresses.v1'
async function fill(page, name, street) {
	await page.getByLabel('Название адреса', { exact: true }).fill(name)
	// React Aria Select opens a listbox; native hidden select is deliberately ignored.
	await page.getByRole('button', { name: /Населённый пункт/ }).click()
	await page.getByRole('option', { name: 'Белореченск', exact: true }).click()
	await page.getByLabel('Улица и дом', { exact: true }).fill(street)
	await page.getByLabel('Квартира / офис', { exact: true }).fill('12')
	await page.getByLabel('Подъезд, этаж', { exact: true }).fill('2')
}
async function save(page) {
	await page.getByRole('button', { name: 'Сохранить', exact: true }).click()
	await page.getByRole('status').filter({ hasText: 'Адрес сохранён' }).waitFor()
}
async function remove(page, card) {
	await card.getByRole('button', { name: 'Удалить', exact: true }).click()
	await page.getByRole('dialog').getByRole('button', { name: 'Удалить', exact: true }).click()
	await page.getByRole('dialog').waitFor({ state: 'hidden' })
}
const pending = []
try {
	// Anonymous and expired contexts never offer import, even with guest addresses.
	// Abort Telegram's SDK here: this is not an authenticated native launch proof.
	const anonymous = await browser.newContext()
	await anonymous.addInitScript(() => {
		const entry = {
			name: 'Дом',
			city: 'Белореченск',
			street: 'Гоголя, 61',
			apartment: '12',
			entrance: '2',
			id: crypto.randomUUID(),
			revision: crypto.randomUUID(),
			isDefault: true,
		}
		localStorage.setItem(
			'citypark.addresses.v1',
			JSON.stringify({ version: 1, revision: crypto.randomUUID(), entries: [entry] })
		)
	})
	const anonymousPage = await anonymous.newPage()
	await anonymousPage.route('https://telegram.org/js/telegram-web-app.js', (route) => route.abort())
	for (const path of ['/', '/menu', '/checkout', '/contacts', '/mini']) {
		await anonymousPage.goto(base + path)
		await expect(anonymousPage.getByRole('region', { name: 'Импорт адресов' })).toHaveCount(0)
		assert.equal(
			await anonymousPage.evaluate(() => JSON.parse(localStorage.getItem('citypark.addresses.v1')).entries.length),
			1
		)
	}
	await anonymous.addCookies([
		{
			name: '__Host-citypark_customer',
			value: 'cs_invalid_qa',
			domain: new URL(base).hostname,
			path: '/',
			secure: true,
			httpOnly: true,
			sameSite: 'Lax',
		},
	])
	await anonymousPage.goto(base + '/contacts')
	await expect(anonymousPage.getByRole('region', { name: 'Импорт адресов' })).toHaveCount(0)
	await anonymousPage.goto(base + '/mini')
	await expect(anonymousPage.getByRole('region', { name: 'Импорт адресов' })).toHaveCount(0)
	await anonymous.close()
	console.log(
		'PASS import suppressed for anonymous/invalid profile across site/menu/checkout/plain/mini layouts; guest data retained and no cross-surface fallback'
	)

	for (const width of [390, 1440])
		for (const theme of ['light', 'dark']) {
			const context = await browser.newContext({ viewport: { width, height: 1000 } })
			await context.addInitScript((theme) => localStorage.setItem('theme', theme), theme)
			const page = await context.newPage()
			await page.route('**/api/address-suggest?**', (route) => route.abort())
			// privacy: collect everything the page could leak to console, errors or other origins.
			const leaks = []
			const secrets = ['Гоголя', 'Ленина', 'Дом QA', 'Работа QA', 'Дом новый', 'Дубликат']
			const encoded = secrets.map((value) => encodeURIComponent(value))
			const watch = (text) => {
				if (secrets.some((value) => text.includes(value))) leaks.push('console/pageerror')
			}
			page.on('console', (message) => watch(message.text()))
			page.on('pageerror', (error) => watch(String(error.message)))
			page.on('request', (request) => {
				const url = request.url()
				const external = new URL(url).origin !== new URL(base).origin
				const body = request.postData() ?? ''
				if (external && (secrets.some((v) => body.includes(v)) || encoded.some((v) => url.includes(v))))
					leaks.push('external request')
			})
			await page.goto(base + '/account/addresses')
			await expect(page.getByRole('region', { name: 'Импорт адресов' })).toHaveCount(0)
			await page.getByRole('heading', { name: 'Мои адреса', exact: true }).waitFor()
			await page.getByText('У вас пока нет сохранённых адресов', { exact: true }).waitFor()
			const add = page.getByRole('button', { name: 'Добавить адрес', exact: true })
			await add.focus()
			await page.keyboard.press('Enter')
			await page.getByLabel('Название адреса', { exact: true }).waitFor()
			assert.equal(
				await page.getByLabel('Название адреса', { exact: true }).evaluate((el) => el === document.activeElement),
				true
			)
			await fill(page, 'Дом QA', 'ул Гоголя, д 61')
			await save(page)
			await page.waitForFunction(() => document.activeElement?.textContent === 'Добавить адрес')
			const cards = page.locator('main li')
			assert.equal(await cards.count(), 1)
			await cards.first().getByText('Основной адрес', { exact: true }).waitFor()
			await add.click()
			await fill(page, 'Работа QA', 'ул Ленина, д 20')
			await save(page)
			await cards.nth(1).getByRole('button', { name: 'Сделать основным' }).click()
			await cards.nth(1).getByText('Основной адрес', { exact: true }).waitFor()
			await cards.first().getByRole('button', { name: 'Изменить' }).click()
			await page.getByLabel('Название адреса', { exact: true }).fill('Дом новый')
			await page.getByRole('button', { name: 'Отмена', exact: true }).click()
			await cards.first().getByRole('heading', { name: 'Дом QA', exact: true }).waitFor()
			await cards.first().getByRole('button', { name: 'Изменить' }).click()
			await page.getByLabel('Название адреса', { exact: true }).fill('Дом новый')
			await save(page)
			await page.reload()
			await cards.first().getByRole('heading', { name: 'Дом новый', exact: true }).waitFor()
			// Duplicate creation preserves the existing name and count.
			await page.getByRole('button', { name: 'Добавить адрес', exact: true }).click()
			await fill(page, 'Дубликат', 'ул Гоголя, д 61')
			await save(page)
			assert.equal(await cards.count(), 2)
			await cards.first().getByRole('button', { name: 'Изменить' }).click()
			await page.getByLabel('Улица и дом', { exact: true }).fill('ул Ленина, д 20')
			await page.getByRole('button', { name: 'Сохранить', exact: true }).click()
			await page.getByRole('status').filter({ hasText: 'Такой адрес уже сохранён' }).waitFor()
			assert.equal(await page.getByLabel('Улица и дом', { exact: true }).inputValue(), 'ул Ленина, д 20')
			await page.getByRole('button', { name: 'Отмена', exact: true }).click()
			await cards.first().getByRole('button', { name: 'Удалить', exact: true }).click()
			await page.getByRole('dialog').getByRole('button', { name: 'Отмена', exact: true }).click()
			assert.equal(await cards.count(), 2)
			await remove(page, cards.first())
			await remove(page, cards.first())
			await page.getByText('У вас пока нет сохранённых адресов', { exact: true }).waitFor()
			assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
			assert.deepEqual(leaks, [], `privacy: saved-address text leaked (${width}px ${theme})`)
			console.log(`PASS privacy: no address text in console, page errors or external requests: ${width}px ${theme}`)
			// responsiveA11y: every visible management control is at least 44 px tall.
			const small = await page.evaluate(() =>
				[...document.querySelectorAll('main button, main a, main input')]
					.filter((el) => el.offsetParent !== null && el.getBoundingClientRect().height < 44)
					.map((el) => (el.getAttribute('aria-label') ?? el.textContent ?? el.tagName).trim().slice(0, 30))
			)
			assert.deepEqual(small, [], `responsiveA11y: controls under 44 px (${width}px ${theme})`)
			console.log(`PASS responsiveA11y: management controls >= 44 px: ${width}px ${theme}`)
			await page.evaluate((key) => localStorage.setItem(key, '{broken'), key)
			await page.reload()
			await page.getByRole('button', { name: 'Сбросить сохранённые адреса', exact: true }).click()
			await page.getByRole('dialog').getByRole('button', { name: 'Отмена', exact: true }).click()
			assert.equal(await page.evaluate((key) => localStorage.getItem(key), key), '{broken')
			await page.getByRole('button', { name: 'Сбросить сохранённые адреса', exact: true }).click()
			await page.getByRole('dialog').getByRole('button', { name: 'Удалить', exact: true }).click()
			await page.getByText('У вас пока нет сохранённых адресов', { exact: true }).waitFor()
			await page.evaluate((key) => localStorage.setItem(key, '{"version":999}'), key)
			await page.reload()
			await page.getByText('Обновите страницу, чтобы открыть сохранённые адреса', { exact: true }).waitFor()
			assert.equal(await page.getByRole('button', { name: 'Добавить адрес', exact: true }).isDisabled(), true)
			assert.equal(await page.evaluate((key) => localStorage.getItem(key), key), '{"version":999}')
			console.log(
				`PASS expected guest CRUD, duplicate/conflict, default, cancel, reload, focus, reset/read-only, no overflow: ${width}px ${theme}`
			)
			// Task 57: guest-only storage writes and invalid contact fields. Never create an order.
			const home = {
				id: randomUUID(),
				revision: randomUUID(),
				isDefault: true,
				name: 'Дом',
				city: 'Белореченск',
				street: 'Гоголя, 61',
				apartment: '12',
				entrance: '2',
			}
			const work = {
				...home,
				id: randomUUID(),
				revision: randomUUID(),
				isDefault: false,
				name: 'Работа',
				street: 'Ленина, 20',
				apartment: '',
				entrance: '4',
			}
			await page.evaluate(
				({ key, entries, cart }) => {
					localStorage.setItem(key, JSON.stringify({ version: 1, revision: crypto.randomUUID(), entries }))
					localStorage.setItem('citypark-cart', JSON.stringify(cart))
				},
				{ key, entries: [home, work], cart: checkoutCart }
			)
			await page.goto(base + '/checkout')
			const selector = page.locator('select[name="savedAddress"]')
			const choose = async (key) => {
				await page.getByRole('button', { name: /Сохранённые адреса/ }).click()
				await page
					.getByRole('option', {
						name: key === 'guest:' + home.id ? 'Дом' : key ? 'Работа' : 'Другой адрес',
						exact: true,
					})
					.click()
			}
			const street = page.getByRole('combobox', { name: 'Улица и дом', exact: true })
			const apartment = page.locator('input[name="apartment"]'),
				entrance = page.locator('input[name="entrance"]')
			await expect(selector).toHaveValue('guest:' + home.id)
			await expect(street).toHaveValue(home.street)
			await expect(apartment).toHaveValue('12')
			await expect(entrance).toHaveValue('2')
			await expect(page.getByLabel('Сохранить адрес', { exact: true })).not.toBeChecked()
			await choose('guest:' + work.id)
			await expect(street).toHaveValue(work.street)
			await expect(apartment).toHaveValue('')
			await expect(entrance).toHaveValue('4')
			await apartment.fill('15')
			await expect(selector).toHaveValue('')
			await expect(street).toHaveValue(work.street)
			await expect(entrance).toHaveValue('4')
			await choose('guest:' + home.id)
			await choose('')
			await expect(street).toHaveValue('')
			await expect(apartment).toHaveValue('')
			await expect(entrance).toHaveValue('')
			await choose('guest:' + home.id)
			// Simulate another tab deleting the selection; its delivery draft survives.
			await page.evaluate((key) => {
				const book = JSON.parse(localStorage.getItem(key))
				book.entries = book.entries.slice(1)
				localStorage.setItem(key, JSON.stringify(book))
				dispatchEvent(new StorageEvent('storage', { key }))
			}, key)
			await expect(selector).toHaveValue('')
			await expect(street).toHaveValue(home.street)
			await page.getByLabel('Сохранить адрес', { exact: true }).check()
			await page.getByLabel('Название адреса', { exact: true }).fill('Новый дом')
			const submit = page.getByRole('button', { name: /^Оплатить/ }).filter({ visible: true })
			await expect(submit).toBeEnabled({ timeout: 10000 })
			// Missing name, phone and consent guarantee validation stops order dispatch.
			await submit.click()
			await page.waitForFunction((key) => JSON.parse(localStorage.getItem(key)).entries.length === 2, key)
			await expect(page.getByText('Укажите имя', { exact: true })).toBeVisible()
			await expect(street).toHaveValue(home.street)
			await expect(apartment).toHaveValue('12')
			await expect(entrance).toHaveValue('2')
			await expect(page.getByLabel('Название адреса', { exact: true })).toHaveValue('Новый дом')
			await submit.click()
			await expect(page.getByText('Укажите имя', { exact: true })).toBeVisible()
			assert.equal(await page.evaluate((key) => JSON.parse(localStorage.getItem(key)).entries.length, key), 2)
			await page.evaluate((key) => {
				const original = Storage.prototype.setItem
				Storage.prototype.setItem = function (k, v) {
					if (k === key) throw Error('quota')
					return original.call(this, k, v)
				}
			}, key)
			await street.fill('Совсем новая, 5')
			await submit.click()
			await expect(page.getByText('Не удалось сохранить адрес в этом браузере', { exact: true })).toBeVisible()
			await expect(street).toHaveValue('Совсем новая, 5')
			await page.getByLabel('Сохранить адрес', { exact: true }).uncheck()
			await page.getByText('Самовывоз', { exact: true }).click()
			await expect(page.getByLabel('Сохранить адрес', { exact: true })).toHaveCount(0)
			assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
			console.log(
				`PASS checkout default/selection/units/removal/error retention, explicit save/duplicate/storage failure/pickup: ${width}px ${theme}`
			)

			await context.close()
		}
	for (const surface of ['web', 'mini']) {
		const token = process.env[surface === 'web' ? 'QA_WEB_SESSION' : 'QA_MINI_SESSION']
		if (!token || process.env.QA_ADDRESS_WRITES !== '1') {
			pending.push(`${surface} authenticated address mutations: session and QA_ADDRESS_WRITES=1 required`)
			continue
		}
		const context = await browser.newContext({ viewport: { width: 390, height: 1000 } })
		await context.addCookies([
			{
				name: surface === 'web' ? '__Host-citypark_customer' : '__Host-citypark_mini',
				value: token,
				domain: new URL(base).hostname,
				path: '/',
				secure: true,
				httpOnly: true,
				sameSite: 'Lax',
			},
		])
		const page = await context.newPage()
		await page.route('**/api/address-suggest?**', (route) => route.abort())
		await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
			route.fulfill({
				contentType: 'application/javascript',
				body: 'window.Telegram={WebApp:{initData:"qa",ready(){},expand(){}}}',
			})
		)
		const path = surface === 'web' ? '/account/addresses' : '/mini/addresses'
		const name = `QA ${Date.now()}`
		let previousDefault
		await page.goto(base + path)
		previousDefault = await page
			.locator('main li')
			.filter({ has: page.getByText('Основной адрес', { exact: true }) })
			.getByRole('heading')
			.textContent()
			.catch(() => null)
		await page.getByRole('button', { name: 'Добавить адрес', exact: true }).click()
		await fill(page, name, `ул Тестовая, д ${Date.now()}`)
		await save(page)
		const card = page.locator('main li').filter({ has: page.getByRole('heading', { name, exact: true }) })
		try {
			await page.reload()
			await card.waitFor()
			assert.equal(await page.evaluate((key) => localStorage.getItem(key), key), null)
			await card.getByRole('button', { name: 'Изменить' }).click()
			await page.getByLabel('Квартира / офис', { exact: true }).fill('15')
			await save(page)
			if (await card.getByRole('button', { name: 'Сделать основным' }).count())
				await card.getByRole('button', { name: 'Сделать основным' }).click()
			await card.getByText('Основной адрес', { exact: true }).waitFor()
			console.log(`PASS ${surface} owned persistence/edit/default and no guest storage`)
		} finally {
			await remove(page, card)
			if (previousDefault) {
				const original = page
					.locator('main li')
					.filter({ has: page.getByRole('heading', { name: previousDefault, exact: true }) })
				if (await original.getByRole('button', { name: 'Сделать основным' }).count())
					await original.getByRole('button', { name: 'Сделать основным' }).click()
			}
			await context.close()
		}
	}
	for (const surface of ['web', 'mini']) {
		const token = process.env[surface === 'web' ? 'QA_WEB_SESSION' : 'QA_MINI_SESSION']
		if (!token) {
			pending.push(`${surface} import offer/session/consent: existing owner-provided session required`)
			continue
		}
		for (const width of [390, 1440])
			for (const theme of ['light', 'dark']) {
				const context = await browser.newContext({ viewport: { width, height: 1000 } })
				const cookie = {
					name: surface === 'web' ? '__Host-citypark_customer' : '__Host-citypark_mini',
					value: token,
					domain: new URL(base).hostname,
					path: '/',
					secure: true,
					httpOnly: true,
					sameSite: 'Lax',
				}
				await context.addCookies([cookie])
				await context.addInitScript((theme) => localStorage.setItem('theme', theme), theme)
				const page = await context.newPage()
				await page.route('**/api/address-suggest?**', (route) => route.fulfill({ json: [] }))
				await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
					route.fulfill({
						contentType: 'application/javascript',
						body: 'window.Telegram={WebApp:{initData:"qa",ready(){},expand(){}}}',
					})
				)
				const path = surface === 'web' ? '/account/addresses' : '/mini/addresses'
				const readBook = async () => {
					const response = await fetch(`${process.env.MENU_API_URL}/customer/addresses`, {
						headers: { Authorization: `Bearer ${token}` },
					})
					assert.equal(response.status, 200)
					return (await response.json()).addresses
				}
				const before = await readBook(),
					beforeIds = new Set(before.map((entry) => entry.id))
				const unique = Date.now() + '-' + width + '-' + theme
				const fields = (n) => ({
					name: `Импорт QA ${n} ${unique}`,
					city: 'Белореченск',
					street: `Тестовая, ${n}-${unique}`,
					apartment: '12',
					entrance: '2',
				})
				const entries = [0, 1].map((n) => ({
					...fields(n),
					id: randomUUID(),
					revision: randomUUID(),
					isDefault: n === 1,
				}))
				const seed = async (entries) =>
					page.evaluate(
						({ key, entries }) => {
							localStorage.setItem(key, JSON.stringify({ version: 1, revision: crypto.randomUUID(), entries }))
							dispatchEvent(new StorageEvent('storage', { key }))
						},
						{ key, entries }
					)
				const region = page.getByRole('region', { name: 'Импорт адресов' })
				const imported = []
				try {
					await page.goto(base + path)
					await expect(region).toHaveCount(0)
					await seed(entries)
					await expect(region).toBeVisible()
					assert.equal(await region.count(), 1)
					await region.getByRole('button', { name: 'Не сейчас', exact: true }).click()
					await expect(region).toHaveCount(0)
					assert.equal(await page.evaluate((key) => JSON.parse(localStorage.getItem(key)).entries.length, key), 2)
					await page.reload()
					await expect(region).toHaveCount(0)
					const manual = page.getByRole('button', { name: 'Добавить адреса из этого браузера', exact: true })
					await manual.click()
					await expect(region).toBeFocused()
					await region.getByRole('button', { name: 'Не сейчас', exact: true }).click()
					await expect(manual).toBeFocused()
					// A new book revision allows another automatic prompt in this same session.
					await seed(entries)
					await expect(region).toBeVisible()
					await context.clearCookies()
					await page.evaluate(() => dispatchEvent(new Event('focus')))
					await expect(region).toHaveCount(0)
					assert.equal(await page.evaluate((key) => JSON.parse(localStorage.getItem(key)).entries.length, key), 2)
					await context.addCookies([cookie])
					await page.goto(base + path)
					await expect(region).toBeVisible()
					// Layout navigation must not require password-login-specific hooks.
					if (surface === 'web')
						for (const target of ['/menu', '/checkout', '/contacts']) {
							await page.goto(base + target)
							await expect(region).toBeVisible()
							assert.equal(await region.count(), 1)
						}
					await page.goto(base + path)
					await expect(region).toBeVisible()
					assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
					console.log(
						`PASS real-session ${surface} import prompt/dismiss/manual focus/revision/logout/navigation: ${width}px ${theme}`
					)
					if (process.env.QA_IMPORT_WRITES !== '1' || process.env.QA_ADDRESS_WRITES !== '1') {
						pending.push(
							`${surface} import writes ${width}px ${theme}: owner must set QA_IMPORT_WRITES=1 and QA_ADDRESS_WRITES=1`
						)
						continue
					}
					await region.getByRole('button', { name: 'Добавить адреса', exact: true }).focus()
					await page.keyboard.press('Enter')
					await expect(region.getByText('Добавлено: 2. Уже сохранено: 0.', { exact: true })).toBeVisible({
						timeout: 20000,
					})
					await expect(region.getByRole('button', { name: 'Закрыть', exact: true })).toBeVisible()
					let after = await readBook()
					const importedDefaultId = after.find((entry) => entry.isDefault)?.id
					imported.push(
						...after.filter(
							(entry) => !beforeIds.has(entry.id) && entries.some((guest) => guest.street === entry.street)
						)
					)
					assert.equal(imported.length, 2)
					if (before.length)
						assert.equal(after.find((entry) => entry.isDefault)?.id, before.find((entry) => entry.isDefault)?.id)
					else assert.equal(after.find((entry) => entry.isDefault)?.street, entries[1].street)
					assert.equal(await page.evaluate((key) => JSON.parse(localStorage.getItem(key)).entries.length, key), 0)
					await region.getByRole('button', { name: 'Закрыть', exact: true }).click()
					await seed([{ ...entries[0], name: 'Другое гостевое название', revision: randomUUID() }])
					await expect(region).toBeVisible()
					await region.getByRole('button', { name: 'Добавить адреса', exact: true }).click()
					await expect(region.getByText('Добавлено: 0. Уже сохранено: 1.', { exact: true })).toBeVisible({
						timeout: 20000,
					})
					after = await readBook()
					assert.equal(after.find((entry) => entry.id === imported[0].id).name, imported[0].name)
					assert.equal(after.find((entry) => entry.isDefault)?.id, importedDefaultId)
					console.log(
						`PASS authorized ${surface} sequential import/counts/cleanup/duplicate/default preservation: ${width}px ${theme}`
					)
				} finally {
					if (process.env.QA_IMPORT_WRITES === '1' && process.env.QA_ADDRESS_WRITES === '1') {
						// Discover committed rows even after a lost UI response, then remove only our fixtures.
						const current = await readBook()
						const ours = current.filter(
							(entry) => !beforeIds.has(entry.id) && entries.some((guest) => guest.street === entry.street)
						)
						await page.goto(base + path)
						if (await region.isVisible()) await region.getByRole('button', { name: /Не сейчас|Закрыть/ }).click()
						for (const entry of ours) {
							const card = page
								.locator('main li')
								.filter({ has: page.getByRole('heading', { name: entry.name, exact: true }) })
							await remove(page, card)
						}
					}
					await context.close()
				}
			}
	}
	pending.push(
		'Task 58 blocker: actual password/Google return-to-page and native Telegram relaunch/import/device acceptance require owner verification; isolated tests and supplied sessions are not provider/device acceptance'
	)

	pending.push(
		'Second-account isolation, legacy/unavailable seeded fixtures, expired/changed account browser responses and native Telegram acceptance require supplied accounts/device'
	)
	// Named acceptance matrix (task 59). Each row states what ran in this process, in isolated harnesses
	// or backend tests, and what only an owner session/device can prove. Blocked rows are never passes.
	const authorized = process.env.QA_ADDRESS_WRITES === '1' && !!process.env.QA_WEB_SESSION
	const matrix = [
		[
			'guestCRUD',
			'PASS',
			'this flow + address-store/management harnesses (CRUD, reload, corrupt/unknown version, quota)',
		],
		[
			'accountCRUD',
			authorized ? 'RAN' : 'BLOCKED',
			'needs QA_ADDRESS_WRITES=1 + QA_WEB_SESSION/QA_OTHER_WEB_SESSION/QA_MINI_SESSION; isolation logic: address-api harness',
		],
		[
			'sharedLocationSafety',
			'PARTIAL',
			'backend tests TestBookOwnerIsolation/TestDeleteEntry/TestBookConcurrentCreates pass; live two-account run blocked',
		],
		[
			'legacyReview',
			'BLOCKED',
			'needs seeded legacy account fixtures; UI logic covered by address-management harness only',
		],
		['checkoutPrefill', 'PASS', 'this flow + checkout-address harness + checkout-geolocation flow'],
		['checkoutSave', 'PASS', 'this flow + checkout-address harness (sequencing); no order or payment created'],
		[
			'loginImport',
			'PARTIAL',
			'address-import harness + anonymous/invalid-session checks; real login/Google/Telegram return blocked',
		],
		['sessionChanges', 'PARTIAL', 'address-import/api harnesses; expiry/account switch with real sessions blocked'],
		['privacy', 'PASS', 'this flow: console, page errors, external requests; address-store/api harness privacy checks'],
		[
			'responsiveA11y',
			'PASS',
			'this flow: 390/1440 px, light/dark, focus, 44 px targets, no overflow; native Telegram keyboard blocked',
		],
	]
	for (const [name, status, note] of matrix) console.log(`${status.padEnd(8)} ${name}: ${note}`)
	console.log('BLOCKED acceptance: ' + pending.join('; '))
	if (process.env.QA_ADDRESS_REQUIRE_ALL === '1') process.exitCode = 1
} finally {
	await browser.close()
}
