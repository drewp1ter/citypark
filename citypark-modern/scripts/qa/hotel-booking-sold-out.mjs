// One real isolated winner, one stale-quote loser; recover that rejection in four view/theme states.
import assert from 'node:assert/strict'
import { randomBytes, randomUUID } from 'node:crypto'
import { mkdir, writeFile } from 'node:fs/promises'
import { chromium } from 'playwright'
import env from '@next/env'
env.loadEnvConfig(process.cwd())
assert.equal(process.env.HOTEL_BOOKING_QA_APPROVED, '1')
assert.ok(process.env.HOTEL_BOOKING_QA_EMAIL, 'Provide an approved development email')
const base = process.env.BASE_URL ?? 'http://localhost:3111'
const stay = {
	roomTypeId: 'single-balcony',
	checkIn: process.env.HOTEL_BOOKING_QA_CHECK_IN ?? '2027-04-18',
	checkOut: process.env.HOTEL_BOOKING_QA_CHECK_OUT ?? '2027-04-20',
	guests: 1,
}
const url = base + '/booking?' + new URLSearchParams({ ...stay, guests: String(stay.guests) })
const token = randomBytes(32).toString('base64url')
const key = randomUUID()
const browser = await chromium.launch({ executablePath: process.env.BROWSER_PATH })
const fixturePath = '/tmp/citypark-hotel-booking-qa/sold-out-winner.json'
async function api(path, body) {
	const response = await fetch(process.env.MENU_API_URL + '/hotel' + path, {
		method: body === undefined ? 'GET' : 'POST',
		headers: { 'Content-Type': 'application/json', 'X-Hotel-Booking-Context': token },
		body: body === undefined ? undefined : JSON.stringify(body),
		signal: AbortSignal.timeout(25000),
		redirect: 'error',
	})
	return { status: response.status, body: await response.json() }
}
let savedContext, savedSession, rejectedUrl, winner
try {
	for (const [index, [width, theme]] of [
		[390, 'light'],
		[390, 'dark'],
		[1440, 'light'],
		[1440, 'dark'],
	].entries()) {
		const context = await browser.newContext({
			viewport: { width, height: 960 },
			colorScheme: theme,
			storageState: savedContext,
		})
		await context.addInitScript(
			({ theme, session }) => {
				localStorage.setItem('theme', theme)
				for (const [key, value] of Object.entries(session ?? {})) sessionStorage.setItem(key, value)
			},
			{ theme, session: savedSession }
		)
		const page = await context.newPage()
		const errors = []
		page.on('pageerror', (error) => errors.push(error.message))
		await page.route('**/*', (route) =>
			route.request().headers()['next-router-prefetch'] ? route.abort() : route.continue()
		)
		await page.goto(rejectedUrl ?? url)
		if (index === 0) {
			await page.getByRole('button', { name: 'Проверить наличие и стоимость', exact: true }).click()
			await page.getByRole('button', { name: 'Забронировать', exact: true }).waitFor()
			await page.getByLabel('ФИО', { exact: true }).fill('Тест отказа сайта 73')
			await page.getByLabel('Электронная почта (необязательно)', { exact: true }).fill(process.env.HOTEL_BOOKING_QA_EMAIL)
			await page.getByLabel('Телефон', { exact: true }).fill('+70000000073')
			const quote = await api('/quotes', stay)
			assert.equal(quote.status, 200)
			await mkdir('/tmp/citypark-hotel-booking-qa', { recursive: true, mode: 0o700 })
			// Persist the winner identity before dispatch; do not create a replacement on timeout.
			await writeFile(fixturePath, JSON.stringify({ key, context: token, stay, quoteId: quote.body.quoteId }), {
				mode: 0o600,
			})
			let created = await api('/reservations', {
				idempotencyKey: key,
				quoteId: quote.body.quoteId,
				guest: { name: 'Тест занятости сайта 73', email: process.env.HOTEL_BOOKING_QA_EMAIL, phone: '+70000000073' },
			})
			for (let i = 0; created.status === 202 && i < 12; i++) {
				await new Promise((resolve) => setTimeout(resolve, 2500))
				created = await api('/reservations/' + key)
			}
			assert.ok([200, 201].includes(created.status), 'Winner must confirm; retain saved key on uncertain outcome')
			assert.equal(created.body.status, 'confirmed')
			winner = created.body
			await writeFile(fixturePath, JSON.stringify({ key, context: token, stay, result: winner }), { mode: 0o600 })
			await page.getByRole('button', { name: 'Забронировать', exact: true }).click()
		}
		await page
			.getByRole('alert')
			.filter({ hasText: 'На выбранные даты нет свободного номера' })
			.waitFor({ timeout: 30000 })
		assert.equal(await page.getByLabel('ФИО', { exact: true }).inputValue(), 'Тест отказа сайта 73')
		assert.equal(await page.locator('[name=checkIn]').inputValue(), stay.checkIn)
		assert.equal(await page.locator('[name=checkOut]').inputValue(), stay.checkOut)
		assert.equal(await page.getByRole('button', { name: 'Забронировать', exact: true }).count(), 0)
		assert.equal(await page.getByRole('heading', { name: 'Бронирование подтверждено', exact: true }).count(), 0)
		assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false)
		assert.deepEqual(errors, [])
		if (index === 0) {
			rejectedUrl = page.url()
			assert.ok(new URL(rejectedUrl).searchParams.get('attempt'))
			savedContext = await context.storageState()
			savedSession = await page.evaluate(() => Object.fromEntries(Object.entries(sessionStorage)))
		}
		await page.screenshot({ path: `/tmp/citypark-hotel-booking-qa/sold-out-${width}-${theme}.png`, fullPage: true })
		console.log(`PASS sold-out ${width}px ${theme}`)
		await context.close()
	}
	console.log(
		JSON.stringify({
			winnerKey: key,
			reservationId: winner.reservationId,
			...stay,
			price: winner.price,
			rejectedKey: new URL(rejectedUrl).searchParams.get('attempt'),
		})
	)
} finally {
	await browser.close()
}
