// Production browser QA. No backend writes in the default readonly/keyboard stages.
// Other stages need explicit approval and an isolated Go -> 1C development target:
// HOTEL_BOOKING_QA_APPROVED=1 HOTEL_BOOKING_QA_STAGE=quote ... node scripts/qa/flows/hotel-booking.mjs
// Set HOTEL_BOOKING_QA_CHECK_IN/CHECK_OUT/ROOM/GUESTS. Creation also requires NAME/EMAIL/PHONE.
// Scenario setup and cleanup: docs/HOTEL_BOOKING.md. No mocks or automatic price/inventory mutations.
import assert from 'node:assert/strict'
import { chromium } from 'playwright'
import { mkdir, chmod, writeFile } from 'node:fs/promises'

const base = process.env.BASE_URL ?? 'http://localhost:3111'
const stage = process.env.HOTEL_BOOKING_QA_STAGE ?? 'readonly'
const stages = [
	'readonly',
	'search',
	'guest-validation',
	'quote',
	'confirm',
	'double-submit',
	'sold-out',
	'price-change',
	'pending-recovery',
	'outage-preservation',
	'keyboard',
]
assert.ok(stages.includes(stage), 'Unknown booking QA stage')
const readonly = stage === 'readonly' || stage === 'keyboard'
if (!readonly)
	assert.equal(
		process.env.HOTEL_BOOKING_QA_APPROVED,
		'1',
		'Obtain owner approval for the isolated development target before quote/booking writes'
	)
const stay = {
	roomTypeId: process.env.HOTEL_BOOKING_QA_ROOM ?? 'single-balcony',
	checkIn: process.env.HOTEL_BOOKING_QA_CHECK_IN ?? '2027-04-10',
	checkOut: process.env.HOTEL_BOOKING_QA_CHECK_OUT ?? '2027-04-12',
	guests: process.env.HOTEL_BOOKING_QA_GUESTS ?? '1',
}
const params = new URLSearchParams(stay)
if (stage === 'pending-recovery') {
	assert.ok(
		process.env.HOTEL_BOOKING_QA_ATTEMPT,
		'Set a known pending key and load its browser context with HOTEL_BOOKING_QA_STORAGE_STATE'
	)
	assert.ok(process.env.HOTEL_BOOKING_QA_STORAGE_STATE)
	params.set('attempt', process.env.HOTEL_BOOKING_QA_ATTEMPT)
}
const offlineQuote = stage === 'outage-preservation' && process.env.HOTEL_BOOKING_QA_OFFLINE_QUOTE === '1'
const mutating =
	['confirm', 'double-submit', 'sold-out', 'price-change', 'outage-preservation'].includes(stage) && !offlineQuote
if (mutating)
	for (const field of ['NAME', 'EMAIL', 'PHONE'])
		assert.ok(process.env['HOTEL_BOOKING_QA_' + field], 'Set approved synthetic ' + field)
await mkdir('/tmp/citypark-hotel-booking-qa', { recursive: true, mode: 0o700 })
await chmod('/tmp/citypark-hotel-booking-qa', 0o700)
const browser = await chromium.launch({ executablePath: process.env.BROWSER_PATH })
let confirmedContext, confirmedUrl
let droppedCreation = false
let creationArmed = false
let creationRequests = 0
const cleanup = []
try {
	for (const width of [390, 1440])
		for (const theme of ['light', 'dark']) {
			const context = await browser.newContext({
				viewport: { width, height: 960 },
				colorScheme: theme,
				storageState: confirmedContext ?? process.env.HOTEL_BOOKING_QA_STORAGE_STATE,
			})
			await context.addInitScript((theme) => localStorage.setItem('theme', theme), theme)
			const page = await context.newPage()
			const errors = []
			page.on('pageerror', (error) => errors.push(error.message))
			// Avoid unrelated link prefetches; in readonly mode this also prevents search quote persistence.
			await page.route('**/*', async (route) => {
				const request = route.request()
				if (request.headers()['next-router-prefetch'] || (readonly && new URL(request.url()).pathname === '/rooms'))
					return route.abort()
				if (
					creationArmed &&
					request.method() === 'POST' &&
					request.headers()['next-action'] &&
					(request.postData() ?? '').includes('quoteId')
				) {
					creationRequests++
					if (process.env.HOTEL_BOOKING_QA_DROP_CREATE_RESPONSE === '1' && !droppedCreation) {
						const upstream = await route.fetch({ timeout: 60000 })
						const body = await upstream.text()
						assert.ok(
							body.includes('\"status\":\"confirmed\"'),
							'The dropped response must contain the committed confirmation'
						)
						droppedCreation = true
						return route.abort('failed')
					}
				}
				return route.continue()
			})
			const response = await page.goto(confirmedUrl ?? base + '/booking?' + params)
			assert.equal(response.status(), 200)
			assert.match(await page.locator('meta[name=robots]').getAttribute('content'), /noindex/)
			assert.match(response.headers()['cache-control'], /no-store/)
			assert.equal(await page.locator('h1').count(), 1)
			if (!confirmedUrl && stage !== 'pending-recovery') {
				assert.equal(await page.locator('[name=checkIn]').inputValue(), stay.checkIn)
				assert.equal(await page.locator('[name=checkOut]').inputValue(), stay.checkOut)
				assert.equal(await page.locator('select[name=guests]').inputValue(), stay.guests)
			}
			if (stage === 'search') {
				await page.goto(base + '/rooms?' + params)
				const link = page
					.locator('article')
					.filter({ has: page.locator(`a[href^="/rooms/${stay.roomTypeId}?"]`) })
					.getByRole('link', { name: /Выбрать|Проверить|Забронировать/ })
					.first()
				const href = await link.getAttribute('href')
				const target = new URL(href, base)
				for (const [key, value] of Object.entries(stay)) assert.equal(target.searchParams.get(key), value)
				await link.click()
				await page.waitForURL('**/booking?**')
			}

			if (offlineQuote) {
				const name = 'Тест недоступности сайта 73'
				await page.getByLabel('ФИО', { exact: true }).fill(name)
				await page.getByLabel('Электронная почта', { exact: true }).fill('signup@retand.tech')
				await page.getByLabel('Телефон', { exact: true }).fill('+70000000073')
				await context.setOffline(true)
				await page.getByRole('button', { name: 'Проверить наличие и стоимость', exact: true }).click()
				await page.getByRole('alert').filter({ hasText: 'Бронирование временно недоступно' }).waitFor()
				assert.equal(await page.getByLabel('ФИО', { exact: true }).inputValue(), name)
				assert.equal(await page.getByLabel('Электронная почта', { exact: true }).inputValue(), 'signup@retand.tech')
				assert.equal(await page.locator('[name=checkIn]').inputValue(), stay.checkIn)
				assert.equal(await page.locator('[name=checkOut]').inputValue(), stay.checkOut)
				await context.setOffline(false)
				await page.getByRole('button', { name: 'Проверить наличие и стоимость', exact: true }).click()
				await page.getByRole('button', { name: 'Забронировать', exact: true }).waitFor()
				assert.equal(await page.getByLabel('ФИО', { exact: true }).inputValue(), name)
			}
			if (
				[
					'quote',
					'guest-validation',
					...['confirm', 'double-submit', 'sold-out', 'price-change', 'outage-preservation'],
				].includes(stage) &&
				!confirmedUrl &&
				!offlineQuote
			) {
				await page.getByRole('button', { name: 'Проверить наличие и стоимость', exact: true }).click()
				await page.getByRole('button', { name: 'Забронировать', exact: true }).waitFor()
				assert.match(await page.locator('form').innerText(), /Итого:/)
				if (stage === 'guest-validation') {
					await page.getByRole('button', { name: 'Забронировать', exact: true }).click()
					await page.getByRole('alert').filter({ hasText: 'Проверьте данные бронирования' }).waitFor()
					assert.equal(await page.locator('[aria-invalid=true]').count(), 3)
				}
				if (mutating) {
					const originalTotal = await page
						.locator('form')
						.getByText(/^Итого:/)
						.innerText()
					await page.getByLabel('ФИО', { exact: true }).fill(process.env.HOTEL_BOOKING_QA_NAME)
					await page.getByLabel('Электронная почта', { exact: true }).fill(process.env.HOTEL_BOOKING_QA_EMAIL)
					await page.getByLabel('Телефон', { exact: true }).fill(process.env.HOTEL_BOOKING_QA_PHONE)
					if (stage === 'price-change' && process.env.HOTEL_BOOKING_QA_AUTOMATIC_PRICE_CHANGE === '1') {
						const { changeRoomRate } = await import('../hotel-booking-admin.mjs')
						cleanup.push(await changeRoomRate(stay.roomTypeId))
					} else if (['sold-out', 'price-change', 'outage-preservation'].includes(stage)) {
						// The operator changes isolated inventory/rate or stops the upstream after the quote.
						assert.equal(
							process.env.HOTEL_BOOKING_QA_INTERACTIVE,
							'1',
							'Use interactive mode to arrange the real race/change/outage after the quote'
						)
						console.log(
							'Arrange the approved ' +
								stage +
								' condition, then press Enter. No automatic backend mutation is performed.'
						)
						await new Promise((resolve) => process.stdin.once('data', resolve))
					}
					const button = page.getByRole('button', { name: 'Забронировать', exact: true })
					creationArmed = true
					if (stage === 'double-submit')
						await button.evaluate((button) => {
							button.click()
							button.click()
						})
					else await button.click()
					if (stage === 'sold-out' || stage === 'price-change') {
						const expected =
							stage === 'sold-out'
								? 'На выбранные даты нет свободного номера'
								: 'Стоимость проживания изменилась. Подтвердите новую стоимость'
						await page.getByRole('alert').filter({ hasText: expected }).waitFor({ timeout: 30000 })
						assert.equal(await page.getByLabel('ФИО', { exact: true }).inputValue(), process.env.HOTEL_BOOKING_QA_NAME)
						assert.equal(await page.getByRole('button', { name: 'Забронировать', exact: true }).count(), 0)
						if (stage === 'price-change' && cleanup.length) {
							const requestsBeforeRefresh = creationRequests
							creationArmed = false
							await page.getByRole('button', { name: 'Проверить наличие и стоимость', exact: true }).click()
							await page.getByRole('button', { name: 'Забронировать', exact: true }).waitFor()
							assert.match(await page.locator('form').innerText(), /Итого:/)
							assert.equal(
								await page.getByLabel('ФИО', { exact: true }).inputValue(),
								process.env.HOTEL_BOOKING_QA_NAME
							)
							assert.notEqual(
								await page
									.locator('form')
									.getByText(/^Итого:/)
									.innerText(),
								originalTotal,
								'Show the changed price for explicit consent'
							)
							assert.equal(creationRequests, requestsBeforeRefresh, 'Refreshing a price must not create a booking')
							await cleanup.pop()()
						}
					} else if (stage === 'outage-preservation') {
						await page.getByRole('heading', { name: 'Проверяем бронирование' }).waitFor({ timeout: 30000 })
						const attempt = new URL(page.url()).searchParams.get('attempt')
						assert.ok(attempt)
						await page.reload()
						await page.getByRole('heading', { name: 'Проверяем бронирование' }).waitFor()
						assert.equal(new URL(page.url()).searchParams.get('attempt'), attempt)
						assert.equal(await page.getByLabel('ФИО', { exact: true }).inputValue(), process.env.HOTEL_BOOKING_QA_NAME)
					} else {
						await page.getByRole('heading', { name: 'Бронирование подтверждено' }).waitFor({ timeout: 60000 })
						creationArmed = false
						if (stage === 'double-submit')
							assert.equal(creationRequests, 1, 'A double click must dispatch one creation action')
						if (process.env.HOTEL_BOOKING_QA_DROP_CREATE_RESPONSE === '1') assert.equal(droppedCreation, true)
						confirmedUrl = page.url()
						confirmedContext = await context.storageState()
						console.log(
							'Created one development fixture; keep its browser context for recovery and arrange staff cancellation afterward.'
						)
						await context.storageState({ path: '/tmp/citypark-hotel-booking-qa/recovery-context.json' })
						await chmod('/tmp/citypark-hotel-booking-qa/recovery-context.json', 0o600)
						await writeFile(
							'/tmp/citypark-hotel-booking-qa/fixture.json',
							JSON.stringify({ url: confirmedUrl, stay, stage, droppedCreation, creationRequests }),
							{ mode: 0o600 }
						)
					}
				}
			}
			if (confirmedUrl || stage === 'pending-recovery') {
				await page.getByRole('heading', { name: 'Бронирование подтверждено' }).waitFor({ timeout: 60000 })
				const attempt = new URL(page.url()).searchParams.get('attempt')
				await page.reload()
				await page.getByRole('heading', { name: 'Бронирование подтверждено' }).waitFor()
				assert.equal(new URL(page.url()).searchParams.get('attempt'), attempt)
			}
			if (readonly) {
				await page.getByLabel('ФИО', { exact: true }).fill('Тест сохранения')
				await page.getByLabel('Электронная почта', { exact: true }).fill('draft@example.invalid')
				await page.getByRole('button', { name: 'Гости', exact: false }).last().click()
				await page.getByRole('option', { name: '4', exact: true }).click()
				assert.equal(await page.getByLabel('ФИО', { exact: true }).inputValue(), 'Тест сохранения')
				await page.getByLabel('ФИО', { exact: true }).focus()
				await page.keyboard.press('Tab')
				assert.equal(
					await page.getByLabel('Электронная почта', { exact: true }).evaluate((el) => el === document.activeElement),
					true
				)
			}

			if (stage === 'keyboard') {
				await page.locator('form button').first().focus()
				const visited = new Set()
				for (let step = 0; step < 22; step++) {
					const focused = await page.evaluate(() => {
						const element = document.activeElement
						if (!element?.closest('form')) return null
						const style = getComputedStyle(element)
						const pickerRing =
							element instanceof HTMLInputElement &&
							element.type === 'date' &&
							!element.matches(':focus-visible') &&
							[...document.styleSheets].some((sheet) => {
								try {
									const containsPickerRule = (rules) =>
										[...rules].some(
											(rule) =>
												rule.cssText.includes('calendar-picker-indicator:focus-visible') ||
												(rule.cssRules && containsPickerRule(rule.cssRules))
										)
									return containsPickerRule(sheet.cssRules)
								} catch {
									return false
								}
							})

						return {
							pickerRing,
							tag: element.tagName,
							name: element.getAttribute('name'),
							focusVisible: element.matches(':focus-visible'),
							submit: element.getAttribute('type') === 'submit',
							outline: parseFloat(style.outlineWidth),
							outlineStyle: style.outlineStyle,
						}
					})
					if (focused) {
						if (focused.name) visited.add(focused.name)
						if (focused.submit) visited.add('submit')
						// Programmatic initial focus can follow pointer input; Tab must reveal the ring.
						if (step > 0)
							assert.ok(
								focused.pickerRing || (focused.outline > 0 && focused.outlineStyle !== 'none'),
								'Keyboard focus must remain visible: ' + JSON.stringify(focused)
							)
					}
					await page.keyboard.press('Tab')
				}
				for (const name of ['checkIn', 'checkOut', 'name', 'email', 'phone', 'submit'])
					assert.ok(visited.has(name), 'Keyboard reaches ' + name)
				const guests = page.getByRole('button', { name: 'Гости', exact: false }).last()
				await guests.focus()
				await page.keyboard.press('Space')
				await page.keyboard.press('ArrowUp')
				await page.keyboard.press('Enter')
				assert.equal(await page.locator('select[name=guests]').inputValue(), '3')
			}
			assert.equal(
				await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
				false,
				'Horizontal overflow'
			)
			assert.deepEqual(errors, [])
			await page.screenshot({ path: `/tmp/citypark-hotel-booking-qa/${stage}-${width}-${theme}.png`, fullPage: true })
			console.log(`PASS ${stage} ${width}px ${theme}`)
			await context.close()
		}
} finally {
	for (const restore of cleanup.reverse()) await restore()
	await browser.close()
}
