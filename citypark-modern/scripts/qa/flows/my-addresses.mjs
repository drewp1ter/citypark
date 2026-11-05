// Production build: npm run build && npm start -- -p 3111
// Guest checks need no backend writes. Backend at MENU_API_URL must already run.
// Account acceptance (owner authorization required): provide existing disposable customer
// sessions QA_WEB_SESSION, QA_OTHER_WEB_SESSION, QA_MINI_SESSION. Seed legacy/unavailable
// addresses on these accounts before running to verify their full raw text/correction.
// Authorized run: QA_ADDRESS_WRITES=1 QA_WEB_SESSION=... QA_OTHER_WEB_SESSION=...
// QA_MINI_SESSION=... node --env-file=.env scripts/qa/flows/my-addresses.mjs
// This flow creates/edits/defaults/deletes temporary addresses only. No orders or payments.
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
	for (const width of [390, 1440])
		for (const theme of ['light', 'dark']) {
			const context = await browser.newContext({ viewport: { width, height: 1000 } })
			await context.addInitScript((theme) => localStorage.setItem('theme', theme), theme)
			const page = await context.newPage()
			await page.route('**/api/address-suggest?**', (route) => route.abort())
			await page.goto(base + '/account/addresses')
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
	pending.push(
		'Second-account isolation, legacy/unavailable seeded fixtures, expired/changed account browser responses and native Telegram acceptance require supplied accounts/device'
	)
	console.log('BLOCKED acceptance: ' + pending.join('; '))
	if (process.env.QA_ADDRESS_REQUIRE_ALL === '1') process.exitCode = 1
} finally {
	await browser.close()
}
