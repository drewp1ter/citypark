import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const base = (process.env.BASE_URL ?? 'http://localhost:3111').replace(/\/+$/, '')
const browser = await chromium.launch({ headless: true })
const context = await browser.newContext()
const page = await context.newPage()

try {
	await page.goto(`${base}/account/sign-in`)
	await page.getByRole('button', { name: 'Войти через Google' }).waitFor()
	await page.getByRole('link', { name: 'Забыли пароль?' }).click()
	await page.waitForURL(`${base}/account/recovery`)
	await page.getByRole('heading', { name: 'Восстановление пароля' }).waitFor()
	assert.equal(await page.getByLabel('Email').getAttribute('type'), 'email')
	console.log('Recovery page and Google entry: PASS')

	for (const path of ['reset', 'email']) {
		const response = await page.goto(`${base}/account/${path}?token=qa-read-only-proof`)
		assert.equal(response.headers()['referrer-policy'], 'no-referrer')
		assert.match(response.headers()['cache-control'], /no-store/)
		assert.equal(await page.locator('input[name=token]').inputValue(), 'qa-read-only-proof')
		assert.equal(await page.locator('form').count(), 1)
		console.log(`${path} GET renders confirmation form without mutation: PASS`)
	}
	await page.goto(`${base}/account/reset?token=qa-read-only-proof`)
	await page.getByLabel('Новый пароль').focus()
	assert.equal(await page.evaluate(() => document.activeElement?.getAttribute('name')), 'password')
	for (const theme of ['light', 'dark']) {
		await page.evaluate((value) => document.documentElement.setAttribute('data-theme', value), theme)
		assert.equal(await page.getByRole('button', { name: 'Сменить пароль' }).isVisible(), true)
	}
	console.log('Reset keyboard focus and both themes: PASS')

	for (const suffix of ['', '?state=invalid&code=fake', '?state=invalid&error=access_denied']) {
		const response = await context.request.get(`${base}/account/google/callback${suffix}`, { maxRedirects: 0 })
		assert.equal(response.status(), 303)
		assert.equal(new URL(response.headers().location).pathname, '/account/sign-in')
		assert.equal(new URL(response.headers().location).searchParams.get('google'), 'failed')
		assert.match(response.headers()['cache-control'], /no-store/)
		assert.equal(response.headers()['referrer-policy'], 'no-referrer')
	}
	console.log('Google callback without browser proof fails closed: PASS')

	await page.goto(`${base}/account/sign-in?google=cancelled`)
	await page.getByRole('alert').getByText('Вход через Google отменён').waitFor()
	await page.goto(`${base}/account/sign-in?google=failed`)
	await page.getByRole('alert').getByText('Не удалось подтвердить вход через Google').waitFor()
	console.log('Google cancellation and failure messages: PASS')

	if (process.env.QA_CUSTOMER_WRITES === '1') {
		const email = process.env.QA_CUSTOMER_EMAIL
		assert.ok(email, 'QA_CUSTOMER_EMAIL is required for approved recovery writes')
		await page.goto(`${base}/account/recovery`)
		await page.getByLabel('Email').fill(email)
		await page.getByRole('button', { name: 'Отправить ссылку' }).click()
		await page.getByText('Если аккаунт найден, письмо для восстановления отправлено').waitFor()
		console.log('Generic recovery response: PASS')
		await page.goto(`${base}/account/reset?token=qa-invalid-proof`)
		await page.getByLabel('Новый пароль').fill('qa-secret-password-123')
		await page.getByRole('button', { name: 'Сменить пароль' }).click()
		await page.getByRole('alert').getByText('Ссылка недействительна или устарела').waitFor()
		assert.ok(!(await page.locator('body').innerHTML()).includes('qa-secret-password-123'))
		console.log('Invalid reset and password non-echo: PASS')
		console.log('Reset, email confirmation, and live Google login require owner-supplied single-use proofs and Google consent.')
		if (process.env.QA_GOOGLE_START === '1') {
			await page.goto(`${base}/account/sign-in`)
			const provider = page.waitForRequest((request) => request.url().startsWith('https://accounts.google.com/o/oauth2/v2/auth?'))
			await page.getByRole('button', { name: 'Войти через Google' }).click()
			const url = new URL((await provider).url())
			assert.deepEqual(url.searchParams.get('scope').split(' ').sort(), ['email', 'openid', 'profile'])
			assert.equal(url.searchParams.get('code_challenge_method'), 'S256')
			assert.ok(url.searchParams.get('state'))
			const cookie = (await context.cookies()).find((item) => item.name === '__Host-citypark_google_flow')
			assert.ok(cookie?.httpOnly && cookie.secure && cookie.sameSite === 'Lax' && cookie.path === '/')
			assert.ok(!url.toString().includes('flowToken'))
			console.log('Google start URL, scopes, PKCE, and browser proof cookie: PASS')
		} else {
			console.log('Google start skipped: set QA_GOOGLE_START=1 when credentials are configured.')
		}
	} else {
		console.log('Backend writes skipped: set QA_CUSTOMER_WRITES=1 after owner approval.')
	}
} finally {
	await browser.close()
}
