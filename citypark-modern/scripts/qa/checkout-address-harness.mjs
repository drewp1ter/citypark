// Save-before-order is explicit opt-in: an acknowledged address survives a later
// order/payment failure. These isolated dependencies dispatch no live orders,
// payments, notifications or address writes. Browser/local storage and login
// changes are not an atomic transaction. Live account acceptance needs owner access.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import vm from 'node:vm'
import ts from 'typescript'
const require = createRequire(import.meta.url)
function load(path, modules) {
	const exports = {}
	const source = ts.transpileModule(readFileSync(path, 'utf8'), {
		compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
	}).outputText
	vm.runInNewContext(source, {
		exports,
		require: (name) => (name in modules ? modules[name] : require(name)),
		queueMicrotask,
		console: { error() {} },
		FormData,
		Date,
	})
	return exports
}
const schemas = load('src/entities/address/model/schema.ts', {})
const first = {
	id: '1',
	name: 'Дом',
	city: 'Белореченск',
	street: 'Гоголя, 61',
	apartment: '12',
	entrance: '2',
	isDefault: true,
}
const second = {
	...first,
	id: '2',
	name: 'Работа',
	street: 'Ленина, 20',
	apartment: '',
	entrance: '4',
	isDefault: false,
}
const towns = [{ name: first.city }]
const guestBook = { context: { kind: 'guest', surface: 'web' }, entries: [] }
let snapshot, localCalls, localFailure
function hook(book = guestBook) {
	snapshot = { status: 'loading', book: { entries: [] } }
	localCalls = []
	localFailure = false
	const cells = [],
		deps = [],
		effects = []
	let cursor = 0
	const react = {
		useState(initial) {
			const i = cursor++
			if (!(i in cells)) cells[i] = initial
			return [
				cells[i],
				(value) => {
					cells[i] = typeof value === 'function' ? value(cells[i]) : value
				},
			]
		},
		useRef(value) {
			const i = cursor++
			return (cells[i] ??= { current: value })
		},
		useCallback(fn) {
			return fn
		},
		useEffect(fn, next) {
			const i = cursor++
			if (!deps[i] || next.some((v, j) => v !== deps[i][j])) effects.push(fn)
			deps[i] = next
		},
	}
	const { useCheckoutAddress: runHook } = load('src/widgets/food-ordering/model/use-checkout-address.ts', {
		react,
		'@/entities/address': {
			...schemas,
			useGuestAddresses: (enabled) => {
				assert.equal(enabled, book.context.kind === 'guest')
				return snapshot
			},
			saveGuestAddress: async (fields) => {
				localCalls.push(fields)
				return localFailure ? { ok: false, message: 'Storage failed' } : { ok: true }
			},
		},
	})
	function render() {
		cursor = 0
		return runHook(book, towns)
	}
	async function settle() {
		for (let i = 0; i < 3; i++) {
			render()
			effects.splice(0).forEach((fn) => fn())
			await Promise.resolve()
		}
		return render()
	}
	return { render, settle }
}
let h = hook(),
	a = h.render()
assert.equal(a.ready, false)
assert.equal(a.locationAllowed, false)
snapshot = { status: 'ready', book: { entries: [first, second] } }
a = await h.settle()
assert.equal(a.selected, 'guest:1')
assert.equal(a.locationAllowed, false)
assert.equal(
	JSON.stringify(a.fields),
	JSON.stringify({ city: first.city, street: first.street, apartment: '12', entrance: '2' })
)
a.select('guest:2')
a = h.render()
assert.equal(a.fields.apartment, '')
assert.equal(a.fields.entrance, '4')
a.edit('street', 'Manual')
a = h.render()
assert.equal(a.selected, '')
assert.equal(a.fields.entrance, '4')
a.detected('Late town', 'Late street')
assert.equal(h.render().fields.street, 'Manual')
a.edit('city', 'Another')
a = h.render()
assert.equal(a.fields.street + a.fields.apartment + a.fields.entrance, '')
a.select('guest:1')
a = h.render()
a.select('')
a = h.render()
assert.equal(Object.values(a.fields).join(''), '')
// Manual edits before hydration beat the default and all later storage changes.
h = hook()
a = h.render()
a.edit('street', 'Early edit')
snapshot = { status: 'ready', book: { entries: [first] } }
a = await h.settle()
assert.equal(a.fields.street, 'Early edit')
// Removal deselects without erasing the draft. New defaults never overwrite it.
h = hook()
snapshot = { status: 'ready', book: { entries: [first] } }
a = await h.settle()
snapshot.book.entries = [second]
a = h.render()
assert.equal(a.selected, '')
assert.equal(a.fields.street, first.street)
for (const entry of [
	{ ...first, city: 'Removed town' },
	{ ...first, needsReview: true },
]) {
	h = hook({ context: { kind: 'account', surface: 'mini', accountId: 7 }, entries: [entry] })
	a = await h.settle()
	assert.equal(a.selected, '')
	a.select('account:1')
	assert.equal(h.render().fields.city, '')
}
h = hook({ context: { kind: 'account', surface: 'web', accountId: 7 }, entries: [], error: 'Load failed' })
a = await h.settle()
assert.equal(a.locationAllowed, true)
a.edit('street', 'Still editable')
assert.equal(h.render().fields.street, 'Still editable')
console.log(
	'PASS checkoutAddressPriority/Selection/UnavailableAddresses: hydration, default, manual edits, late detection, exact units, city clearing, removed entries, legacy and failed list'
)
h = hook()
snapshot = { status: 'ready', book: { entries: [first] } }
a = await h.settle()
assert.equal(await a.beforeSubmit('delivery'), true)
assert.equal(localCalls.length, 0)
a.setSave(true)
a = h.render()
assert.equal(await a.beforeSubmit('pickup'), true)
assert.equal(localCalls.length, 0)
assert.equal(await a.beforeSubmit('delivery'), true)
assert.equal(localCalls.length, 1)
localFailure = true
assert.equal(await a.beforeSubmit('delivery'), false)
assert.equal(h.render().fields.street, first.street)
a.edit('street', '')
a = h.render()
const count = localCalls.length
assert.equal(await a.beforeSubmit('delivery'), false)
assert.equal(localCalls.length, count)
console.log(
	'PASS checkoutGuestSave: unchecked/pickup, explicit save, storage and validation failure preserve draft and stop dispatch'
)
class CustomerApiError extends Error {
	constructor(status, message) {
		super(message)
		this.status = status
	}
}
const libs = { isValidRuPhone: () => true, hoursError: () => 'Hours', phoneDigits: (value) => value.replace(/\D/g, '') }
const orderSchema = load('src/widgets/food-ordering/model/schema.ts', { '@/shared/lib': libs })
const transactionId = '12345678-abcd-abcd-abcd-123456789abc'
let context, calls, saveFailure, orderFailure, missingTransactionId
const action = load('src/widgets/food-ordering/api/place-order.ts', {
	'next/navigation': {
		redirect: (path) => {
			throw Object.assign(Error('Redirect'), { path })
		},
	},
	'@/entities/address': { ...schemas, formatAddress: (f) => f.street },
	'@/entities/address/index.server': {
		loadAddressBook: async (surface) => {
			calls.push(['context', surface])
			return { context, entries: [] }
		},
		createSavedAddress: async (...args) => {
			calls.push(['save', ...args])
			if (saveFailure) throw saveFailure
			return { created: false, entry: first }
		},
	},
	'@/entities/dish': {
		getMenu: async () => ({ dishes: [{ id: '1', externalId: '1' }] }),
		getDeliveryTowns: async () => towns,
		priceLines: () => ({ items: [{ dish: { externalId: '1' }, quantity: 1 }] }),
	},
	'@/shared/api': { getWorkingHours: async () => ({ start: '09:00', end: '22:00' }) },
	'@/shared/lib': libs,
	'@/shared/customer': {
		CustomerApiError,
		transactionIdFromRedirect: (url) => new URL(url).searchParams.get('MNT_TRANSACTION_ID'),
		setPendingPaymentContext: async (...args) => calls.push(['payment', ...args]),
	},
	'../model/schema': orderSchema,
	'./orders-api': {
		createOrder: async (...args) => {
			calls.push(['order', ...args])
			return orderFailure
				? { ok: false, message: 'Order failed' }
				: { ok: true, orderId: 8, redirectUrl: `https://demo.moneta.ru/assistant.htm${missingTransactionId ? '' : '?MNT_TRANSACTION_ID=' + transactionId}` }
		},
	},
})
function form(patch = {}) {
	const f = new FormData()
	for (const [k, v] of Object.entries({
		method: 'delivery',
		saveAddress: 'on',
		addressName: 'Дом',
		addressContext: 'account',
		expectedAccountId: '7',
		city: first.city,
		street: first.street,
		apartment: '12',
		entrance: '2',
		name: 'Имя',
		phone: '+79183119791',
		when: 'asap',
		consent: 'on',
		lines: '[{"dishId":"1","variantId":"default","quantity":1}]',
		...patch,
	}))
		f.set(k, v)
	return f
}
for (const surface of ['web', 'mini']) {
	const submit = surface === 'web' ? action.placeOrder : action.placeMiniOrder
	const reset = () => {
		context = { kind: 'account', surface, accountId: 7 }
		calls = []
		saveFailure = null
		orderFailure = false
		missingTransactionId = false
	}
	reset()
	await assert.rejects(submit({}, form()), (e) => e.path?.startsWith('https://demo.moneta.ru'))
	assert.equal(calls.map((c) => c[0]).join(','), 'context,save,order,payment')
	assert.equal(calls[1][1], surface)
	assert.equal(calls[1][2], 7)
	assert.equal(calls[2][2], surface)
	assert.deepEqual(calls[3], ['payment', 8, transactionId, surface])
	reset()
	missingTransactionId = true
	await assert.rejects(submit({}, form()), (e) => e.path === 'https://demo.moneta.ru/assistant.htm')
	assert.equal(calls.some((c) => c[0] === 'payment'), false)
	for (const changed of [
		{ kind: 'auth-required', surface },
		{ kind: 'account', surface, accountId: 8 },
		{ kind: 'guest', surface: 'web' },
	]) {
		reset()
		context = changed
		const result = await submit({}, form())
		assert.ok(result.errors.addressSave)
		assert.equal(calls.length, 1)
		assert.equal(result.values.entrance, '2')
		assert.equal(result.values.addressName, 'Дом')
		assert.equal(result.values.saveAddress, 'on')
	}
	reset()
	saveFailure = new CustomerApiError(502, 'Private')
	assert.ok((await submit({}, form())).errors.addressSave)
	assert.equal(
		calls.some((c) => c[0] === 'order'),
		false
	)
	reset()
	orderFailure = true
	let result = await submit({}, form())
	assert.equal(result.errors.form, 'Order failed')
	assert.equal(calls.filter((c) => c[0] === 'save').length, 1)
	calls = []
	result = await submit({}, form())
	assert.equal(calls.filter((c) => c[0] === 'save').length, 1)
	assert.equal(calls.filter((c) => c[0] === 'order').length, 1)
	for (const patch of [{ saveAddress: '' }, { method: 'pickup' }]) {
		reset()
		await assert.rejects(submit({}, form(patch)), (e) => !!e.path)
		assert.equal(
			calls.some((c) => c[0] === 'save'),
			false
		)
	}
	reset()
	assert.ok((await submit({}, form({ street: '' }))).errors.street)
	assert.equal(calls.length, 0)
	reset()
	assert.ok((await submit({}, form({ addressName: 'x'.repeat(41) }))).errors.addressName)
	assert.equal(calls.length, 0)
	reset()
	context = { kind: 'account', surface, accountId: 7 }
	assert.ok((await submit({}, form({ addressContext: 'guest' }))).errors.addressSave)
	assert.equal(calls.length, 1)
}
calls = []
context = { kind: 'guest', surface: 'web' }
saveFailure = null
orderFailure = false
await assert.rejects(action.placeOrder({}, form({ addressContext: 'guest', expectedAccountId: '' })), (e) => !!e.path)
assert.equal(
	calls.some((c) => c[0] === 'save'),
	false
)
console.log(
	'PASS checkoutAccountSave: surface/owner, expired/changed sessions, validation/save failure stops order, save survives order failure, duplicate retry, unchecked/pickup, guest login race, signed payment context'
)
await import('./address-store-harness.mjs')
await import('./payment-retry-harness.mjs')
