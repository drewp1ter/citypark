// Isolated import contracts; no live authentication, account writes, orders or payments.
// These tests do not replace owner-run password/Google/native Telegram acceptance.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { resolve, dirname } from 'node:path'
import { randomUUID } from 'node:crypto'
import vm from 'node:vm'
import ts from 'typescript'
const require = createRequire(import.meta.url)
const plain = (value) => JSON.parse(JSON.stringify(value))
function load(file, modules = {}, globals = {}) {
	const exports = {}
	const source = ts.transpileModule(readFileSync(file, 'utf8'), {
		compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
	}).outputText
	vm.runInNewContext(source, {
		exports,
		require: (name) =>
			name in modules ? modules[name] : name === '@/shared/lib' ? load('src/shared/lib/phone.ts') : require(name),
		...globals,
	})
	return exports
}
const { createImportCoordinator } = load('src/features/import-addresses/model/import-state.ts')
const base = {
	recipientName: 'Иван',
	recipientPhone: '+79991234567',
	name: 'Дом',
	city: 'Белореченск',
	street: 'Гоголя, 61',
	apartment: '12',
	entrance: '2',
}
function deferred() {
	let resolve
	const promise = new Promise((done) => {
		resolve = done
	})
	return { promise, resolve }
}
async function fixture(surface = 'web', count = 3) {
	const data = new Map(),
		cache = new Map(),
		session = new Map(),
		accounts = new Map()
	let blocked = false,
		blockedSession = false,
		accountId = 7,
		status,
		failureAt,
		loseResponse = false,
		afterSave,
		verifyOverride
	const window = {
		localStorage: {
			getItem: (key) => data.get(key) ?? null,
			setItem: (key, value) => {
				if (blocked) throw Error('quota')
				data.set(key, value)
			},
		},
		addEventListener() {},
		removeEventListener() {},
	}
	function local(file) {
		file = resolve(file)
		if (cache.has(file)) return cache.get(file)
		const value = load(
			file,
			new Proxy(
				{},
				{ has: (_, name) => name.startsWith('.'), get: (_, name) => local(resolve(dirname(file), name + '.ts')) }
			),
			{ window, crypto: { randomUUID } }
		)
		cache.set(file, value)
		return value
	}
	const store = local('src/entities/address/model/guest-store.ts'),
		format = local('src/entities/address/model/format.ts')
	for (let i = 0; i < count; i++)
		assert.equal((await store.saveGuestAddress({ ...base, name: `Гость ${i}`, street: `Гоголя, ${i + 1}` })).ok, true)
	if (count > 1) await store.setGuestDefault(store.getGuestSnapshot().book.entries[1].id)
	const calls = [],
		refreshed = []
	async function importOne(actualSurface, expectedAccountId, fields) {
		calls.push({ surface: actualSurface, accountId: expectedAccountId, fields: { ...fields } })
		assert.equal(actualSurface, surface)
		assert.deepEqual(Object.keys(fields).sort(), [
			'apartment',
			'city',
			'entrance',
			'name',
			'recipientName',
			'recipientPhone',
			'street',
		])
		if (failureAt === calls.length) return { ok: false, kind: 'request', message: 'Request failed' }
		if (expectedAccountId !== accountId)
			return { ok: false, kind: 'auth', message: 'Войдите повторно, чтобы добавить адреса' }
		const book = accounts.get(accountId) ?? []
		accounts.set(accountId, book)
		const key = (f) => format.addressDuplicateKey(f.city, format.formatAddress(f))
		let entry = book.find((entry) => key(entry) === key(fields)),
			created = false
		if (!entry) {
			entry = { ...fields, id: book.length + 1, needsReview: false, isDefault: book.length === 0 }
			book.push(entry)
			created = true
		}
		if (afterSave) await afterSave(entry)
		if (loseResponse) {
			loseResponse = false
			throw Error('Lost response')
		}
		return { ok: true, entry, created }
	}
	const coordinator = createImportCoordinator(surface, {
		guest: () => ({ ...store.getGuestSnapshot(), ...(status ? { status } : {}) }),
		verify: async (s) => {
			assert.equal(s, surface)
			return verifyOverride ? verifyOverride() : accountId
		},
		importOne,
		remove: store.removeImportedGuests,
		storage: () => {
			if (blockedSession) throw Error('blocked session storage')
			return { getItem: (key) => session.get(key) ?? null, setItem: (key, value) => session.set(key, value) }
		},
	})
	const detach = coordinator.attach()
	await coordinator.synchronize()
	return {
		coordinator,
		store,
		calls,
		accounts,
		data,
		session,
		refreshed,
		detach,
		state: () => coordinator.getSnapshot(),
		entries: () => store.getGuestSnapshot().book.entries,
		run: () => coordinator.start(() => refreshed.push('refresh')),
		setAccount: (id) => {
			accountId = id
		},
		block: (value) => {
			blocked = value
		},
		blockSession: (value) => {
			blockedSession = value
		},
		setStatus: (value) => {
			status = value
		},
		failAt: (value) => {
			failureAt = value
		},
		lose: () => {
			loseResponse = true
		},
		after: (fn) => {
			afterSave = fn
		},
		verify: (fn) => {
			verifyOverride = fn
		},
	}
}
// Suppression, revision keys, no automatic writes, manual override, and memory fallback.
let f = await fixture()
assert.equal(f.state().open, true)
assert.equal(f.calls.length, 0)
f.coordinator.offer()
assert.equal(f.state().phase, 'offered')
f.coordinator.dismiss()
f.coordinator.offer()
assert.equal(f.state().open, false)
assert.equal(f.entries().length, 3)
assert.ok([...f.session.keys()][0].includes('web:7:'))
await f.coordinator.synchronize()
assert.equal(f.state().open, false)
f.coordinator.offer(true)
assert.equal(f.state().open, true)
assert.equal(f.state().focus, 1)
f.coordinator.dismiss()
await f.store.saveGuestAddress({ ...base, street: 'Новая, 5' })
f.coordinator.offer()
assert.equal(f.state().open, true)
f.coordinator.dismiss()
f.setAccount(8)
await f.coordinator.synchronize()
assert.equal(f.state().open, true)
f.blockSession(true)
f.coordinator.dismiss()
await f.coordinator.synchronize()
assert.equal(f.state().open, false)
f.coordinator.offer(true)
assert.equal(f.state().open, true)
for (const status of ['loading', 'corrupt', 'unsupported', 'unavailable']) {
	f.coordinator.dismiss()
	f.setStatus(status)
	f.coordinator.offer(true)
	assert.equal(f.state().open, false)
}
f = await fixture('mini', 0)
assert.equal(f.state().open, false)
f.setAccount(null)
await f.coordinator.synchronize()
assert.equal(f.state().accountId, null)
f.verify(() => {
	throw Error('Profile unavailable')
})
await f.coordinator.synchronize()
assert.equal(f.state().open, false)
console.log(
	'PASS importPromptEligibility/importDismissal: known current account only, hydration/empty/errors, one coordinator, account/surface/revision keys, manual override, blocked session storage'
)
// A guest default is attempted first. No default mutation is sent; account metadata wins.
for (const surface of ['web', 'mini'])
	for (const existing of ['empty', 'default', 'no-default', 'legacy-duplicate']) {
		f = await fixture(surface)
		const guestDefault = f.entries().find((e) => e.isDefault)
		if (existing !== 'empty')
			f.accounts.set(7, [
				{
					...guestDefault,
					id: 10,
					name: 'Account metadata',
					isDefault: existing === 'default',
					needsReview: existing === 'legacy-duplicate',
				},
			])
		await f.run()
		assert.equal(f.state().phase, 'success')
		assert.equal(f.calls[0].fields.street, guestDefault.street)
		assert.equal(f.entries().length, 0)
		assert.equal(f.state().created, existing === 'empty' ? 3 : 2)
		assert.equal(f.state().skipped, existing === 'empty' ? 0 : 1)
		const book = f.accounts.get(7)
		assert.equal(book[0].isDefault, existing === 'empty' || existing === 'default')
		if (existing !== 'empty') assert.equal(book[0].name, 'Account metadata')
		assert.equal(f.state().needsReview, existing === 'legacy-duplicate')
		assert.equal(f.refreshed.length, 3)
	}
console.log(
	'PASS importDuplicatesAndDefaults: both surfaces, default first, exact counts, duplicate metadata, nonempty no-default and review warning'
)
for (const failureAt of [1, 2, 3]) {
	f = await fixture()
	f.failAt(failureAt)
	await f.run()
	assert.equal(f.state().phase, 'error')
	assert.equal(f.calls.length, failureAt)
	assert.equal(f.entries().length, 4 - failureAt)
	assert.equal(f.state().created, failureAt - 1)
	f.failAt(undefined)
	await f.run()
	assert.equal(f.state().phase, 'success')
	assert.equal(f.state().created, 4 - failureAt)
	assert.equal(f.state().skipped, 0)
	assert.equal(f.entries().length, 0)
}
f = await fixture()
f.lose()
await f.run()
assert.equal(f.state().phase, 'error')
assert.equal(f.entries().length, 3)
assert.equal(f.accounts.get(7).length, 1)
await f.run()
assert.equal(f.state().created, 2)
assert.equal(f.state().skipped, 1)
f = await fixture()
f.block(true)
await f.run()
assert.equal(f.entries().length, 3)
assert.equal(f.state().created, 1)
assert.equal(f.state().phase, 'error')
f.block(false)
await f.run()
assert.equal(f.entries().length, 0)
assert.equal(f.state().created, 2)
assert.equal(f.state().skipped, 1)
console.log(
	'PASS importPartialFailure: first/middle/last failure, stopped queue, acknowledged cleanup only, lost response, cleanup failure, fresh retry counts/idempotence'
)
f = await fixture()
const changing = f.entries().find((e) => e.isDefault)
let changed = false
f.after(async () => {
	if (changed) return
	changed = true
	await f.store.saveGuestAddress({ ...changing, street: 'Changed during import, 9' }, changing.id)
	await f.store.saveGuestAddress({ ...base, street: 'Created during import, 10' })
})
await f.run()
assert.equal(f.entries().length, 2)
assert.ok(f.entries().some((e) => e.street === 'Changed during import, 9'))
assert.ok(f.entries().some((e) => e.street === 'Created during import, 10'))
assert.equal(f.state().created, 3)
console.log(
	'PASS importConcurrentGuestEdit: real revision-aware guest store retains changed/new rows; never clears storage key'
)
// Hold an actual acknowledged request across context changes. It may have committed,
// but must never clean local rows, refresh, change counts, or send the remaining queue.
for (const mode of ['logout', 'switch', 'detach', 'dismiss']) {
	f = await fixture()
	const waiting = deferred(),
		arrived = deferred()
	f.after(async () => {
		arrived.resolve()
		await waiting.promise
	})
	const run = f.run()
	await arrived.promise
	assert.equal(f.state().phase, 'pending')
	await f.run()
	assert.equal(f.calls.length, 1)
	if (mode === 'detach') f.detach()
	else if (mode === 'dismiss') f.coordinator.dismiss()
	else {
		f.setAccount(mode === 'logout' ? null : 8)
		await f.coordinator.synchronize()
	}
	waiting.resolve()
	await run
	assert.equal(f.entries().length, 3)
	assert.equal(f.calls.length, 1)
	assert.equal(f.refreshed.length, 0)
	assert.equal(f.state().created, 0)
	assert.equal(f.state().message, '')
}
f = await fixture()
f.setAccount(null)
await f.run()
assert.equal(f.calls.length, 0)
assert.equal(f.entries().length, 3)
assert.equal(f.state().phase, 'error')
assert.ok(f.state().message.includes('Войдите повторно'))
// A switch unobserved by layout effects is still caught by fresh post-response verification.
f = await fixture()
f.after(async () => f.setAccount(8))
await f.run()
assert.equal(f.calls.length, 1)
assert.equal(f.entries().length, 3)
assert.equal(f.state().accountId, 8)
assert.equal(f.state().created, 0)
f = await fixture()
const late = deferred()
let checks = 0
f.verify(() => (++checks === 1 ? late.promise : 8))
const old = f.coordinator.synchronize()
await f.coordinator.synchronize()
late.resolve(7)
await old
assert.equal(f.state().accountId, 8)

// Expiry after a committed response still cannot acknowledge or clean a local row.
f = await fixture()
f.after(async () => f.setAccount(null))
await f.run()
assert.equal(f.calls.length, 1)
assert.equal(f.entries().length, 3)
assert.equal(f.state().phase, 'error')
assert.ok(f.state().message.includes('Войдите повторно'))
// Replacing a layout lease invalidates work even before its new verification settles.
f = await fixture()
const oldLease = deferred(),
	atRequest = deferred()
f.after(async () => {
	atRequest.resolve()
	await oldLease.promise
})
const replacingRun = f.run()
await atRequest.promise
const detachReplacement = f.coordinator.attach()
await f.coordinator.synchronize()
oldLease.resolve()
await replacingRun
assert.equal(f.entries().length, 3)
assert.equal(f.state().created, 0)
f.detach()
assert.equal(f.state().accountId, 7)
detachReplacement()
assert.equal(f.state().accountId, null)
console.log(
	'PASS importSessionIsolation: logout/switch/detach/dismiss, expired session, unobserved switch, duplicate starts, out-of-order verification, no stale cleanup/UI/queued request'
)
// Server entrypoints: only one field object; no URLs, credentials, IDs or default flags.
class CustomerApiError extends Error {
	constructor(status, message, fields = {}) {
		super(message)
		this.status = status
		this.fields = fields
	}
}
const schema = load('src/entities/address/model/schema.ts')
let actionCalls = [],
	invalidated = [],
	ownerFailure,
	tokenCalls = [],
	profile = { id: 7, role: 'user' }
const actions = load('src/features/import-addresses/api/actions.ts', {
	'next/cache': { revalidatePath: (path) => invalidated.push(path) },
	'@/entities/address': schema,
	'@/entities/address/index.server': {
		createSavedAddress: async (...args) => {
			actionCalls.push(args)
			if (ownerFailure) throw ownerFailure
			return { entry: { id: 1, needsReview: false }, created: true }
		},
	},
	'@/shared/customer': {
		CustomerApiError,
		getCustomerSessionToken: async () => {
			tokenCalls.push('web')
			return 'web-secret'
		},
		getMiniSessionToken: async () => {
			tokenCalls.push('mini')
			return 'mini-secret'
		},
		customerRequest: async (path, init) => {
			assert.equal(path, '/customer/me')
			assert.equal(init.headers.Authorization, `Bearer ${tokenCalls.at(-1)}-secret`)
			return profile
		},
	},
})
for (const surface of ['web', 'mini']) {
	tokenCalls = []
	assert.equal(await actions.getImportAccountAction(surface), 7)
	assert.deepEqual(tokenCalls, [surface])
	actionCalls = []
	invalidated = []
	assert.equal((await actions.importAddressAction(surface, 7, base)).ok, true)
	assert.equal(actionCalls[0][0], surface)
	assert.equal(actionCalls[0][1], 7)
	assert.deepEqual(plain(actionCalls[0][2]), base)
	assert.deepEqual(invalidated, [surface === 'web' ? '/account/addresses' : '/mini/addresses'])
	for (const fields of [
		null,
		[],
		[base],
		{ ...base, userId: 8 },
		{ ...base, url: 'https://example.com' },
		{ ...base, isDefault: true },
		{ ...base, street: '' },
	]) {
		const n = actionCalls.length
		assert.equal((await actions.importAddressAction(surface, 7, fields)).ok, false)
		assert.equal(actionCalls.length, n)
	}
	for (const id of [0, -1, 1.5, '7', Number.MAX_SAFE_INTEGER + 1])
		assert.equal((await actions.importAddressAction(surface, id, base)).kind, 'auth')
	for (const failure of [
		new CustomerApiError(401, 'Войдите повторно, чтобы сохранить адрес'),
		new CustomerApiError(400, 'Проверьте адрес', { street: 'Укажите улицу и дом' }),
		new Error('Private address data'),
	]) {
		ownerFailure = failure
		const result = await actions.importAddressAction(surface, 7, base)
		assert.equal(result.ok, false)
		assert.equal(JSON.stringify(result).includes('Private'), false)
	}
	ownerFailure = undefined
}
for (profile of [null, {}, { id: -1, role: 'user' }, { id: 7, role: 'admin' }])
	assert.equal(await actions.getImportAccountAction('web'), null)
assert.equal((await actions.importAddressAction('invalid', 7, base)).kind, 'auth')
console.log(
	'PASS importServerAction: one strictly validated object, surface tokens, expected owner, safe typed auth/validation/transport errors and list revalidation'
)
// UI contract checks use isolated state, never a simulated Google/native acceptance pass.
const { renderToStaticMarkup } = require('react-dom/server'),
	React = require('react')
f = await fixture()
const ui = load('src/features/import-addresses/ui/import-addresses.tsx', {
	react: { useEffect() {}, useRef: () => ({ current: null }), useSyncExternalStore: (_, get) => get() },
	'next/navigation': { usePathname: () => '/account/addresses', useRouter: () => ({ refresh() {} }) },
	'@/entities/address': { useGuestAddresses: () => ({ status: 'ready', book: { entries: f.entries() } }) },
	'@/shared/ui': { Button: (props) => React.createElement('button', props) },
	'../api/actions': {},
	'../model/import-state': { createImportCoordinator: () => f.coordinator },
})
let html = renderToStaticMarkup(React.createElement(ui.ImportAddresses, { surface: 'web', initialAccountId: 7 }))
assert.ok(html.includes('Добавить сохранённые адреса в аккаунт?'))
assert.ok(html.includes('aria-live="polite"'))
assert.ok(html.includes('tabindex="-1"'))
f.coordinator.dismiss()
html = renderToStaticMarkup(
	React.createElement(ui.ImportAddresses, { surface: 'web', initialAccountId: 7, manual: true })
)
assert.ok(html.includes('Добавить адреса из этого браузера'))
f.coordinator.offer(true)
const hold = deferred(),
	ready = deferred()
f.after(async () => {
	ready.resolve()
	await hold.promise
})
const running = f.run()
await ready.promise
html = renderToStaticMarkup(React.createElement(ui.ImportAddresses, { surface: 'web', initialAccountId: 7 }))
assert.match(html, /disabled=""/)
assert.ok(html.includes('Добавляем адреса…'))
hold.resolve()
await running
html = renderToStaticMarkup(React.createElement(ui.ImportAddresses, { surface: 'web', initialAccountId: 7 }))
assert.ok(html.includes('Добавлено: 3. Уже сохранено: 0.'))
console.log(
	'PASS importAccessibility: shared automatic/manual state, labeled focusable region, live progress, pending disabled accept, exact numeric success status; browser/device acceptance separate'
)
await import('./address-api-harness.mjs')
