import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import { signIn } from './helpers'

test('groups, search and the sync button', async ({ page }) => {
	await signIn(page)
	await page.getByRole('link', { name: 'Меню' }).first().click()
	await expect(page.getByRole('heading', { name: 'Меню' })).toBeVisible()
	await expect(page.getByText('Названия, цены, состав и фото меняются в iiko')).toBeVisible()

	const dishes = page.getByRole('list').last().getByRole('listitem')
	const groups = page.getByRole('navigation', { name: 'Категории меню' }).getByRole('button')
	await expect(groups.first()).toBeVisible()
	test.skip((await groups.count()) <= 1, 'the menu is empty in the dev database')

	const first = groups.nth(1)
	await first.click()
	await expect(first).toHaveAttribute('aria-pressed', 'true')
	await expect(page).toHaveURL(/category=\d+/)

	await groups.first().click()
	const name = (await dishes.first().locator('.font-medium').first().innerText()).trim()
	const search = page.getByLabel('Поиск блюда')
	await search.fill(name)
	await expect(dishes.filter({ hasText: name }).first()).toBeVisible()
	await expect(page).toHaveURL(/q=/)
	await search.fill(`нет-такого-${Date.now()}`)
	await expect(page.getByText('Блюд не найдено')).toBeVisible()

	const sync = page.getByRole('button', { name: 'Обновить меню из iiko' })
	await sync.click()
	// iiko may be unconfigured in dev: either toast is fine, but the button must come back.
	await expect(
		page.getByText(/Меню обновлено|Обновление уже идёт|Не удалось|Ошибка|Что-то пошло не так/).first()
	).toBeVisible({
		timeout: 30_000,
	})
	await expect(sync).toBeEnabled()
})

test('category ordering schedule saves and survives reload', async ({ page }) => {
	await signIn(page)
	await page.getByRole('link', { name: 'Меню' }).first().click()
	const groups = page.getByRole('navigation', { name: 'Категории меню' }).getByRole('button')
	await expect(groups.nth(1)).toBeVisible()
	await groups.nth(1).click()
	const editor = page.getByRole('region', { name: 'Настройки категории' })
	const start = editor.getByLabel('Начало заказа')
	const end = editor.getByLabel('Окончание заказа')
	const boxes = editor.getByRole('checkbox')
	const originalStart = await start.inputValue()
	const originalEnd = await end.inputValue()
	const originalDays = await boxes.evaluateAll((nodes) => nodes.map((node) => (node as HTMLInputElement).checked))
	async function save() {
		const response = page.waitForResponse(
			(response) => response.url().includes('/ordering-hours') && response.request().method() === 'PUT'
		)
		await editor.getByRole('button', { name: 'Сохранить расписание' }).click()
		expect((await response).status()).toBe(200)
		await expect(page.getByText('Расписание категории сохранено').first()).toBeVisible()
	}
	try {
		await start.fill('10:00')
		await end.fill('20:00')
		for (let i = 0; i < 7; i++) await boxes.nth(i).setChecked(i < 5)
		await save()
		await page.reload()
		await expect(start).toHaveValue('10:00')
		await expect(end).toHaveValue('20:00')
		for (let i = 0; i < 7; i++) await expect(boxes.nth(i)).toBeChecked({ checked: i < 5 })
	} finally {
		await start.fill(originalStart)
		await end.fill(originalEnd)
		for (const [i, checked] of originalDays.entries()) await boxes.nth(i).setChecked(checked)
		await save()
	}
})

test('dish ordering block persists and can be restored', async ({ page }) => {
	test.setTimeout(120000)
	await signIn(page)
	await page.getByRole('link', { name: 'Меню' }).first().click()
	const control = page.getByRole('switch', { name: 'Заблокировать заказ' }).first()
	await expect(control).toBeVisible()
	const original = await control.isChecked()
	try {
		await control.click()
		await expect(control).toBeChecked({ checked: !original })
		await expect(control).toBeEnabled()
		await page.reload()
		await expect(control).toBeChecked({ checked: !original })
		if (process.env.QA_PUBLIC_BASE_URL && !original) {
			const envText = readFileSync('../citypark-modern/.env', 'utf8')
			const menuUrl = envText
				.match(/^MENU_API_URL=(.*)$/m)?.[1]
				?.trim()
				.replace(/^['"]|['"]$/g, '')
			execFileSync(process.execPath, ['scripts/qa/flows/dish-ordering-blocks.mjs'], {
				cwd: '../citypark-modern',
				env: {
					...process.env,
					MENU_API_URL: menuUrl,
					QA_REQUIRE_BLOCKED_DISH: '1',
					BASE_URL: process.env.QA_PUBLIC_BASE_URL,
				},
				stdio: 'inherit',
				timeout: 90000,
			})
		}
	} finally {
		if ((await control.isChecked()) !== original) await control.click()
		await expect(control).toBeChecked({ checked: original })
		await expect(control).toBeEnabled()
	}
})

test('failed dish block preserves saved state', async ({ page }) => {
	await signIn(page)
	await page.getByRole('link', { name: 'Меню' }).first().click()
	const control = page.getByRole('switch', { name: 'Заблокировать заказ' }).first()
	await expect(control).toBeVisible()
	const original = await control.isChecked()
	await page.route('**/products/*/ordering-block', (route) =>
		route.fulfill({
			status: 400,
			contentType: 'application/json',
			body: JSON.stringify({
				statusCode: 400,
				message: 'Не удалось сохранить блокировку',
				errors: { isOrderBlocked: 'Проверьте настройку' },
			}),
		})
	)
	await control.click()
	await expect(control).toBeEnabled()
	await expect(control).toBeChecked({ checked: original })
	await expect(page.getByText('Проверьте настройку')).toBeVisible()
})
