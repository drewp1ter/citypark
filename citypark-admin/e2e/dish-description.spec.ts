import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

// Requires the updated backend. Temporarily edits one dish and restores its original HTML.
// QA_PUBLIC_BASE_URL enables public dish rendering/fallback checks against a fresh production build.
const apiUrl = readFileSync('.env', 'utf8')
	.match(/^VITE_API_URL=(.*)$/m)![1]!
	.trim()
	.replace(/^['"]|['"]$/g, '')
for (const width of [390, 1440]) {
	for (const theme of ['light', 'dark']) {
		test(`full description: ${width}px ${theme}`, async ({ page, request, context }) => {
			await page.setViewportSize({ width, height: 900 })
			await signIn(page)
			const session = await page.evaluate(() => JSON.parse(sessionStorage.getItem('citypark-admin-token')!))
			const products = await request.get(`${apiUrl}/products?pageSize=500`)
			expect(products.ok()).toBeTruthy()
			const dish = (await products.json())[0]
			expect(dish).toHaveProperty('fullDescription')
			const endpoint = `${apiUrl}/products/${dish.id}/full-description`
			const token = { Authorization: `Bearer ${session.token}` }
			await page.goto(`/menu?dish=${dish.id}`)
			await page.evaluate((theme) => document.documentElement.setAttribute('data-theme', theme), theme)
			const editor = page.getByRole('textbox', { name: 'Полное описание', exact: true })
			const save = page.getByRole('button', { name: 'Сохранить описание', exact: true })
			const text = `Тестовое полное описание ${width} ${theme}`
			async function submit() {
				const response = page.waitForResponse((r) => r.url() === endpoint && r.request().method() === 'PUT')
				await save.click()
				expect((await response).status()).toBe(200)
				await expect(save).toBeEnabled()
			}
			const publicPage = process.env.QA_PUBLIC_BASE_URL ? await context.newPage() : undefined
			try {
				await editor.fill(text)
				await submit()
				await page.reload()
				await expect(editor).toHaveText(text)
				if (publicPage) {
					await publicPage.goto(`${process.env.QA_PUBLIC_BASE_URL}/menu/${dish.id}`)
					await expect(publicPage.getByText(text, { exact: true })).toBeVisible()
					await publicPage.goto(`${process.env.QA_PUBLIC_BASE_URL}/menu`)
					await expect(publicPage.getByText(text, { exact: true })).toHaveCount(0)
				}
				await page.route(endpoint, (route) => route.abort())
				await editor.fill('Черновик после ошибки')
				await save.click()
				await expect(save).toBeEnabled()
				await expect(editor).toHaveText('Черновик после ошибки')
				await page.unroute(endpoint)
				await editor.fill('')
				await submit()
				await page.reload()
				await expect(editor).toBeEmpty()
				if (publicPage && dish.description.trim()) {
					await publicPage.goto(`${process.env.QA_PUBLIC_BASE_URL}/menu/${dish.id}`)
					await expect(publicPage.getByText(dish.description.trim(), { exact: true })).toBeVisible()
				}
				expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy()
			} finally {
				await page.unroute(endpoint)
				const restored = await request.put(endpoint, {
					headers: token,
					data: { fullDescription: dish.fullDescription },
				})
				expect(restored.status()).toBe(200)
				await publicPage?.close()
			}
		})
	}
}
