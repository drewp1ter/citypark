import { expect, test } from '@playwright/test'

test('production worker provides offline fallback and leaves admin data uncached', async ({ page, context }) => {
	await page.goto('/login')
	await page.evaluate(async () => {
		await navigator.serviceWorker.ready
	})
	await expect.poll(() => page.evaluate(() => Boolean(navigator.serviceWorker.controller))).toBe(true)
	const manifest = await (await page.request.get('/manifest.webmanifest')).json()
	expect(manifest.display).toBe('standalone')
	for (const icon of manifest.icons) expect((await page.request.get(icon.src)).ok()).toBe(true)
	await page.evaluate(async () => {
		await caches.open('citypark-admin-offline-obsolete')
		await caches.open('unrelated-cache')
	})
	await context.setOffline(true)
	await page.goto('/orders?test=offline')
	await expect(page.getByRole('heading', { name: 'Нет подключения к интернету' })).toBeVisible()
	const cached = await page.evaluate(async () => {
		const cache = await caches.open('citypark-admin-offline-v1')
		return (await cache.keys()).map((request) => new URL(request.url).pathname)
	})
	expect(cached).toEqual(['/offline.html'])
	await context.setOffline(false)
	await page.getByRole('link', { name: 'Повторить попытку' }).click()
	await expect(page.getByRole('button', { name: 'Войти' })).toBeVisible()
	// Re-registering after unregistering exercises activation and cache ownership.
	await page.evaluate(async () => {
		const registration = await navigator.serviceWorker.getRegistration()
		await registration?.unregister()
	})
	await page.goto('/login')
	await page.evaluate(async () => {
		await navigator.serviceWorker.ready
	})
	await expect.poll(() => page.evaluate(() => caches.keys())).toEqual(['citypark-admin-offline-v1', 'unrelated-cache'])
})

for (const width of [390, 1440]) {
	for (const colorScheme of ['light', 'dark'] as const) {
		test(`install instructions and prompt dismissal at ${width}px in ${colorScheme}`, async ({ page }) => {
			await page.setViewportSize({ width, height: 900 })
			await page.emulateMedia({ colorScheme })
			await page.addInitScript(() => {
				sessionStorage.setItem(
					'citypark-admin-token',
					JSON.stringify({ token: 'pwa-test', email: 'test@example.com', name: 'Test' })
				)
			})
			await page.route('**/api/v1/**', (route) => route.fulfill({ status: 503, json: { message: 'Offline test' } }))
			await page.goto('/orders')
			if (width === 390) await page.getByRole('button', { name: 'Открыть меню' }).click()
			const button = page.getByRole('button', { name: 'Установить приложение', exact: true }).filter({ visible: true })
			await button.click()
			await expect(page.getByRole('status').filter({ hasText: 'На iPhone' })).toBeVisible()
			await page.evaluate(() => {
				const event = new Event('beforeinstallprompt', { cancelable: true })
				Object.assign(event, { prompt: async () => {}, userChoice: Promise.resolve({ outcome: 'dismissed' }) })
				window.dispatchEvent(event)
			})
			await button.click()
			await expect(button).toBeVisible()
			await page.evaluate(() => window.dispatchEvent(new Event('appinstalled')))
			await expect(button).toHaveCount(0)
		})
	}
}
