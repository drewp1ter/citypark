// Isolated transport failures and contracts; never sends a live mutation.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { resolve, dirname } from 'node:path'
import vm from 'node:vm'
import ts from 'typescript'
const require = createRequire(import.meta.url)
const generic = 'Не удалось сохранить адрес. Попробуйте ещё раз.'
const signIn = 'Войдите повторно, чтобы сохранить адрес'
const loadError = 'Не удалось загрузить адреса. Попробуйте ещё раз.'
const fields = {
	recipientName: 'Иван',
	recipientPhone: '+79991234567',
	name: 'Дом',
	city: 'Белореченск',
	street: 'ул. Гоголя, 61',
	apartment: '12',
	entrance: '2',
}
const entry = {
	...fields,
	id: 1,
	address: 'ул. Гоголя, 61, кв. 12',
	isDefault: true,
	needsReview: false,
	createdAt: '2026-11-05T13:07:41.090029Z',
	updatedAt: '2026-11-05T13:07:41.090029Z',
}
const profile = { id: 7, name: 'Иван', email: '', phone: '', role: 'user' }
const calls = [],
	logs = []
let tokens = { web: 'WEB_SECRET', mini: 'MINI_SECRET' },
	activeSurface = 'web',
	getCounts = { web: 0, mini: 0 }
let reply = { addresses: [entry] },
	profileReply = profile,
	profileStatus = 200,
	status = 200,
	network = false
const cache = new Map()
function load(file) {
	file = resolve(file)
	if (cache.has(file)) return cache.get(file)
	const exports = {}
	const source = ts.transpileModule(readFileSync(file, 'utf8'), {
		compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS },
	}).outputText
	vm.runInNewContext(source, {
		exports,
		console: Object.fromEntries(['log', 'error', 'warn', 'info'].map((key) => [key, (...args) => logs.push(args)])),
		fetch: async (url, init) => {
			calls.push({ url, ...init })
			assert.equal(init.headers.Authorization, `Bearer ${tokens[activeSurface]}`)
			assert.equal(init.cache, 'no-store')
			if (network) throw Error('sensitive network URL/address/token')
			const me = url.endsWith('/customer/me')
			const code = me ? profileStatus : status
			return { status: code, ok: code < 400, json: async () => (me ? profileReply : reply) }
		},
		require: (name) => {
			if (name === 'server-only') return {}
			if (name === '@/shared/config') return { serverEnv: () => ({ MENU_API_URL: 'https://private-backend/api/v1' }) }
			if (name === '@/shared/customer')
				return {
					...load('src/shared/customer/customer.ts'),
					getCustomerSessionToken: async () => {
						getCounts.web++
						return tokens.web
					},
					getMiniSessionToken: async () => {
						getCounts.mini++
						return tokens.mini
					},
				}
			return name.startsWith('.')
				? load(resolve(dirname(file), name + '.ts'))
				: name === '@/shared/lib'
					? load('src/shared/lib/phone.ts')
					: require(name)
		},
	})
	cache.set(file, exports)
	return exports
}
const api = load('src/entities/address/api/server.ts')
const json = (value) => (value === undefined ? undefined : JSON.parse(JSON.stringify(value)))
function reset() {
	calls.length = 0
	tokens = { web: 'WEB_SECRET', mini: 'MINI_SECRET' }
	getCounts = { web: 0, mini: 0 }
	activeSurface = 'web'
	profileReply = profile
	profileStatus = 200
	status = 200
	network = false
	reply = { addresses: [entry] }
}
async function rejects(fn, message, expectedFields) {
	await assert.rejects(fn, (error) => {
		assert.equal(error.message, message)
		if (expectedFields) assert.deepEqual(json(error.fields), expectedFields)
		assert.doesNotMatch(error.message, /SECRET|private-backend|Гоголя/)
		return true
	})
}
for (const surface of ['web', 'mini']) {
	reset()
	activeSurface = surface
	const result = await api.loadAddressBook(surface)
	assert.deepEqual(json(result.context), { kind: 'account', surface, accountId: 7 })
	assert.deepEqual(json(result.entries), [entry])
	assert.equal(getCounts[surface], 1)
	assert.equal(getCounts[surface === 'web' ? 'mini' : 'web'], 0)
	tokens[surface] = undefined
	calls.length = 0
	assert.equal((await api.loadAddressBook(surface)).context.kind, surface === 'web' ? 'guest' : 'auth-required')
	assert.equal(calls.length, 0)
	await rejects(() => api.createSavedAddress(surface, 7, fields), signIn)
	assert.equal(calls.length, 0)
	reset()
	activeSurface = surface
	profileStatus = 401
	profileReply = { message: 'Требуется авторизация' }
	assert.equal((await api.loadAddressBook(surface)).context.kind, 'auth-required')
	await rejects(() => api.createSavedAddress(surface, 7, fields), signIn)
	assert.ok(calls.every((call) => call.url.endsWith('/customer/me')))
	reset()
	activeSurface = surface
	await rejects(() => api.createSavedAddress(surface, 8, fields), signIn)
	assert.equal(calls.length, 1)
}
console.log('addressSurfaceIsolation: PASS (expected isolated sessions and guarded writes)')
for (const [name, args, method, path, response] of [
	['createSavedAddress', [fields], 'POST', '', { entry, created: true }],
	['createSavedAddress', [fields], 'POST', '', { entry, created: false }],
	['updateSavedAddress', [1, fields], 'PUT', '/1', { entry }],
	['setSavedDefault', [1], 'PUT', '/1/default', { entry }],
	['deleteSavedAddress', [1], 'DELETE', '/1', undefined],
]) {
	reset()
	reply = response
	status = method === 'DELETE' ? 204 : 200
	const result = await api[name]('web', 7, ...args)
	assert.deepEqual(json(result), json(response?.created === undefined ? response?.entry : response))
	assert.equal(calls[1].method, method)
	assert.ok(calls[1].url.endsWith('/customer/addresses' + path))
	assert.equal(calls[1].body, method === 'POST' || name === 'updateSavedAddress' ? JSON.stringify(fields) : undefined)
}
reset()
reply = { entry, created: true }
await api.createSavedAddress('web', 7, { ...fields, userId: 123, isDefault: true })
assert.deepEqual(JSON.parse(calls[1].body), fields)
for (const [key, bad] of Object.entries({
	id: 0,
	name: null,
	city: 3,
	address: {},
	street: false,
	apartment: [],
	entrance: null,
	isDefault: 1,
	needsReview: 'yes',
	createdAt: 'yesterday',
	updatedAt: '2026-99-01',
})) {
	reset()
	reply = { entry: { ...entry, [key]: bad }, created: true }
	await rejects(() => api.createSavedAddress('web', 7, fields), generic)
}
for (const key of Object.keys(entry)) {
	reset()
	const broken = { ...entry }
	delete broken[key]
	reply = { entry: broken }
	await rejects(() => api.updateSavedAddress('web', 7, 1, fields), generic)
}
for (const bad of [undefined, null, [], { entry }, { entry, created: 'true' }, { entry: null, created: true }]) {
	reset()
	reply = bad
	await rejects(() => api.createSavedAddress('web', 7, fields), generic)
}
for (const bad of [[], {}, { addresses: null }, { addresses: [null] }, { addresses: [{ ...entry, id: 1.5 }] }]) {
	reset()
	reply = bad
	const result = await api.loadAddressBook('web')
	assert.equal(result.context.kind, 'account')
	assert.equal(result.error, loadError)
}
for (const bad of [{}, { ...profile, id: -1 }, { ...profile, role: 'admin' }, { ...profile, phone: null }]) {
	reset()
	profileReply = bad
	const result = await api.loadAddressBook('web')
	assert.equal(result.context.kind, 'auth-required')
	assert.equal(result.error, loadError)
	await rejects(() => api.createSavedAddress('web', 7, fields), generic)
	assert.ok(calls.every((call) => call.url.endsWith('/customer/me')))
}
reset()
reply = { addresses: [{ ...entry, street: '', needsReview: true }] }
assert.equal((await api.loadAddressBook('web')).entries[0].needsReview, true)
for (const id of [0, -1, 1.5, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1, '1']) {
	reset()
	await rejects(() => api.updateSavedAddress('web', 7, id, fields), 'Не найдено')
	assert.equal(calls.length, 1)
}
console.log('addressAPIContract: PASS (expected verbs, envelopes, safe IDs and validated responses)')
for (const [code, message, expected] of [
	[400, 'Проверьте адрес', 'Проверьте адрес'],
	[409, 'Такой адрес уже сохранён', 'Такой адрес уже сохранён'],
	[401, 'Требуется авторизация', signIn],
	[404, 'Не найдено', 'Не найдено'],
	[503, 'Вход временно недоступен', 'Вход временно недоступен'],
	[500, 'raw sensitive data', generic],
	[400, 'raw address', generic],
]) {
	reset()
	status = code
	reply = { message, errors: { street: 'Такой адрес уже сохранён', secret: 'SECRET' } }
	await rejects(
		() => api.updateSavedAddress('web', 7, 1, fields),
		expected,
		[400, 409].includes(code) && expected !== generic ? { street: 'Такой адрес уже сохранён' } : undefined
	)
}
reset()
network = true
await rejects(() => api.createSavedAddress('web', 7, fields), generic)
assert.equal((await api.loadAddressBook('web')).error, loadError)
reset()
await rejects(() => api.createSavedAddress('web', 7, { ...fields, street: '' }), 'Проверьте адрес', {
	street: 'Укажите улицу и дом',
})
assert.equal(calls.length, 1)
console.log('addressAPIErrors: PASS (expected safe messages, field errors and no failed-guard writes)')
assert.deepEqual(logs, [])
const barrel = readFileSync('src/entities/address/index.ts', 'utf8')
assert.doesNotMatch(barrel, /api\/server|shared\/customer|MENU_API_URL/)
assert.ok(readFileSync('src/entities/address/api/server.ts', 'utf8').startsWith("import 'server-only'"))
console.log('addressAPIPrivacy: PASS (expected no logs or server exports in browser entry)')

// Controlled-field wiring: selection/programmatic values do not signal a manual edit.
const fieldExports = {}
vm.runInNewContext(
	ts.transpileModule(readFileSync('src/entities/address/ui/address-fields.tsx', 'utf8'), {
		compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
	}).outputText,
	{
		exports: fieldExports,
		require: (name) => {
			if (name === '@/shared/ui') return { Select: 'Select', TextField: 'TextField' }
			if (name === './address-field') return { AddressField: 'AddressField' }
			return name === '@/shared/lib' ? load('src/shared/lib/phone.ts') : require(name)
		},
	}
)
let edits = 0,
	changed
const controls = fieldExports.AddressFields({
	values: { ...fields, city: 'Старый город' },
	towns: [{ name: 'Новый город' }],
	suggestionsEnabled: true,
	errors: { street: 'Проверьте адрес' },
	onManualEdit: () => edits++,
	onChange: (value) => {
		changed = value
	},
}).props.children
assert.equal(edits, 0)
assert.ok(controls[0].props.options.some((option) => option.value === 'Старый город'))
assert.equal(controls[1].props.error, 'Проверьте адрес')
controls[1].props.onValueChange('Выбранная улица')
assert.equal(edits, 0)
assert.equal(changed.street, 'Выбранная улица')
controls[1].props.onManualEdit()
assert.equal(edits, 1)
controls[0].props.onValueChange('Новый город')
assert.equal(edits, 2)
assert.equal(changed.street, '')
controls[2].props.onChange({ target: { value: '9' } })
assert.equal(changed.apartment, '9')
assert.equal(edits, 3)
controls[3].props.onChange({ target: { value: '4' } })
assert.equal(changed.entrance, '4')
assert.equal(edits, 4)
console.log(
	'addressFieldRegression: PASS (expected controlled values, retained old town, errors and manual-edit distinction)'
)

// Optional production-browser regression; provider responses are intercepted, backend is read-only.
if (process.env.BASE_URL) {
	const { chromium, expect } = await import('playwright/test')
	const { seedCart } = await import('./menu-cart.mjs')
	const cart = await seedCart([1])
	const browser = await chromium.launch()
	try {
		for (const width of [390, 1440]) {
			const context = await browser.newContext({ viewport: { width, height: 950 } })
			await context.addInitScript((cart) => {
				localStorage.setItem('citypark-cart', JSON.stringify(cart))
				Object.defineProperty(navigator, 'geolocation', {
					value: { getCurrentPosition: (_ok, fail) => fail({ code: 1 }) },
				})
			}, cart)
			await context.route('**/api/address-suggest?**', (route) => route.fulfill({ json: ['ул Гоголя, д 61'] }))
			const page = await context.newPage()
			await page.goto(process.env.BASE_URL + '/checkout')
			const street = page.getByRole('combobox', { name: 'Улица и дом' })
			await expect(page.getByRole('status').filter({ hasText: 'Проверьте населённый пункт' })).toBeVisible()
			await street.fill('Гоголя')
			await expect(page.getByRole('option', { name: 'ул Гоголя, д 61' })).toBeVisible()
			await street.press('ArrowDown')
			await street.press('Enter')
			await expect(street).toHaveValue('ул Гоголя, д 61')
			await street.fill('Произвольная улица')
			await expect(page.getByRole('option', { name: 'ул Гоголя, д 61' })).toBeVisible()
			await street.press('Escape')
			await expect(page.getByRole('option', { name: 'ул Гоголя, д 61' })).not.toBeVisible()
			await expect(street).toHaveValue('Произвольная улица')
			await context.route('**/api/address-suggest?**', (route) => route.fulfill({ status: 503, json: {} }))
			await street.fill('Ручной адрес')
			await page.waitForTimeout(850)
			await expect(street).toHaveValue('Ручной адрес')
			await context.close()
			console.log(
				`addressFieldRegression browser ${width}px: PASS (keyboard selection, Escape and failed-provider manual fallback)`
			)
		}
	} finally {
		await browser.close()
	}
}
