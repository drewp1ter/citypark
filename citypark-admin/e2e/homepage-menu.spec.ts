import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import type { components } from '../src/shared/api/schema'
import { signIn } from './helpers'

type Configuration = components['schemas']['HomepageMenu']
type Category = components['schemas']['Category']
type Product = components['schemas']['Product']
const env = readFileSync('.env', 'utf8')
const api = env
	.match(/^VITE_API_URL=(.*)$/m)![1]!
	.trim()
	.replace(/^['"]|['"]$/g, '')
	.replace(/\/$/, '')
test.describe.configure({ mode: 'serial' })

for (const width of [390, 1440])
	for (const theme of ['light', 'dark']) {
		test(`homepage selections ${width} ${theme}`, async ({ page }) => {
			await page.setViewportSize({ width, height: 1000 })
			await page.addInitScript((theme) => localStorage.setItem('citypark-admin-theme', theme), theme)
			await signIn(page)
			const token = await page.evaluate(
				() => (JSON.parse(sessionStorage.getItem('citypark-admin-token')!) as { token: string }).token
			)
			const headers = { Authorization: `Bearer ${token}` }
			const response = await page.request.get(api + '/homepage-menu')
			expect(response.ok(), 'backend92 must be deployed').toBe(true)
			const original: Configuration = await response.json()
			const categories: Category[] = await (await page.request.get(api + '/categories')).json()
			const products: Product[] = []
			for (let pageNumber = 1; ; pageNumber++) {
				const batch: Product[] = await (
					await page.request.get(api + `/products?pageSize=500&page=${pageNumber}`)
				).json()
				products.push(...batch)
				if (batch.length < 500) break
			}
			// Refuse to overwrite an already stale configuration, which cannot be restored through PUT.
			for (const selection of original.categories) {
				expect(categories.some((c) => c.id === selection.categoryId)).toBe(true)
				for (const id of selection.productIds)
					expect(products.some((p) => p.id === id && p.categoryId === selection.categoryId)).toBe(true)
			}
			const category = categories.find((c) => products.filter((p) => p.categoryId === c.id).length >= 4)!
			const other = categories.find((c) => c.id !== category.id && products.some((p) => p.categoryId === c.id))!
			expect(category).toBeDefined()
			expect(other).toBeDefined()
			const dishes = products.filter((p) => p.categoryId === category.id).slice(0, 4)
			const editor = page.getByRole('region', { name: 'Блюда на главной', exact: true })
			const save = editor.getByRole('button', { name: 'Сохранить блюда', exact: true })
			page.on('dialog', (dialog) => dialog.accept())
			async function persist() {
				const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().endsWith('/homepage-menu'))
				await save.click()
				expect((await saved).ok()).toBe(true)
				await expect(save).toBeDisabled()
			}
			try {
				await page.goto('/content/home')
				await expect(editor).toBeVisible()
				await editor.getByRole('button', { name: 'Вернуть автоматический выбор' }).click()
				await editor.getByLabel('Добавить категорию').selectOption(String(category.id))
				let group = editor.getByRole('region', { name: category.name, exact: true })
				for (const dish of dishes.slice(0, 3))
					await group.getByRole('checkbox', { name: dish.name, exact: true }).check()
				await expect(group.getByRole('checkbox', { name: dishes[3]!.name, exact: true })).toBeDisabled()
				await group.getByRole('button', { name: `Поднять блюдо: ${dishes[2]!.name}`, exact: true }).click()
				await expect(group.getByRole('listitem')).toHaveCount(3)
				await group.getByRole('searchbox').fill('НичегоТакогоНет')
				await expect(group.getByText('Блюда не найдены')).toBeVisible()
				await group.getByRole('searchbox').fill('')
				await editor.getByLabel('Добавить категорию').selectOption(String(other.id))
				await editor.getByRole('button', { name: `Поднять категорию: ${other.name}`, exact: true }).click()
				await persist()
				await page.reload()
				await expect(editor).toBeVisible()
				group = editor.getByRole('region', { name: category.name, exact: true })
				const saved: Configuration = await (await page.request.get(api + '/homepage-menu')).json()
				expect(saved).toEqual({
					categories: [
						{ categoryId: other.id, productIds: [] },
						{ categoryId: category.id, productIds: [dishes[0]!.id, dishes[2]!.id, dishes[1]!.id] },
					],
				})
				await expect(group.getByRole('listitem').nth(1)).toContainText(dishes[2]!.name)
				await group.getByRole('button', { name: `Убрать блюдо: ${dishes[1]!.name}`, exact: true }).click()
				await expect(editor.getByText('Есть несохранённые изменения')).toBeVisible()
				await page.evaluate(() => window.dispatchEvent(new Event('visibilitychange')))
				await expect(group.getByRole('listitem')).toHaveCount(2)
				await page.route('**/homepage-menu', async (route) => {
					if (route.request().method() === 'PUT')
						await route.fulfill({
							status: 400,
							json: {
								statusCode: 400,
								message: 'Ошибка проверки',
								errors: { 'categories[1].productIds': 'Ошибка выбора блюд' },
							},
						})
					else await route.continue()
				})
				await save.click()
				await expect(group.getByText('Ошибка выбора блюд')).toBeVisible()
				await expect(group.getByRole('listitem')).toHaveCount(2)
				await page.unroute('**/homepage-menu')
				let release!: () => void
				const gate = new Promise<void>((resolve) => {
					release = resolve
				})
				await page.route('**/homepage-menu', async (route) => {
					if (route.request().method() === 'PUT') {
						await gate
						await route.abort()
					} else await route.continue()
				})
				await save.click()
				await expect(save).toBeDisabled()
				await expect(editor.getByLabel('Добавить категорию')).toBeDisabled()
				release()
				await expect(save).toBeEnabled()
				await expect(group.getByRole('listitem')).toHaveCount(2)
				await page.unroute('**/homepage-menu')
				await persist()
				expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
			} finally {
				await page.unroute('**/homepage-menu')
				const restored = await page.request.put(api + '/homepage-menu', { headers, data: original })
				expect(restored.ok(), 'restore original configuration').toBe(true)
			}
		})
	}

test('homepage editor retries loading and identifies stale choices', async ({ page }) => {
	await signIn(page)
	await page.route('**/homepage-menu', (route) => route.abort())
	await page.goto('/content/home')
	await expect(page.getByRole('button', { name: 'Повторить загрузку блюд' })).toBeVisible({ timeout: 20000 })
	await page.unroute('**/homepage-menu')
	await page.route('**/homepage-menu', async (route) => {
		const response = await route.fetch()
		expect(response.ok()).toBe(true)
		await route.fulfill({ response, json: { categories: [{ categoryId: 999999, productIds: [999999] }] } })
	})
	await page.getByRole('button', { name: 'Повторить загрузку блюд' }).click()
	await expect(page.getByText('Категория скрыта или удалена. Уберите её из списка.')).toBeVisible()
	await expect(page.getByText('Блюдо удалено или перенесено. Уберите его из списка.')).toBeVisible()
	await page.getByRole('button', { name: 'Убрать категорию', exact: true }).click()
	await page.unroute('**/homepage-menu')
	// A dirty form prompts on client navigation; dismiss preserves its draft.
	page.once('dialog', (dialog) => dialog.dismiss())
	await page.getByRole('link', { name: 'Меню', exact: true }).click()
	await expect(page).toHaveURL(/\/content\/home$/)
})
