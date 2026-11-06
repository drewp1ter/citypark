// Isolated guest-store checks; no backend requests or browser acceptance claims.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { resolve, dirname } from 'node:path'
import { randomUUID } from 'node:crypto'
import vm from 'node:vm'
import ts from 'typescript'
const require = createRequire(import.meta.url)
const KEY = 'citypark.addresses.v1'
function environment() {
	const data = new Map(),
		events = new Set(),
		cache = new Map()
	let readFailure = false,
		writeFailure = false,
		writes = 0
	const window = {
		localStorage: {
			getItem: (key) => {
				if (readFailure) throw Error('blocked')
				return data.get(key) ?? null
			},
			setItem: (key, value) => {
				if (writeFailure) throw Error('quota')
				writes++
				data.set(key, value)
			},
		},
		addEventListener: (_, fn) => events.add(fn),
		removeEventListener: (_, fn) => events.delete(fn),
	}
	function load(file) {
		file = resolve(file)
		if (cache.has(file)) return cache.get(file)
		const exports = {}
		const source = ts.transpileModule(readFileSync(file, 'utf8'), {
			compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS },
		}).outputText
		vm.runInNewContext(source, {
			exports,
			window,
			crypto: { randomUUID },
			require: (name) =>
				name.startsWith('.')
					? load(resolve(dirname(file), name + '.ts'))
					: name === '@/shared/lib'
						? load('src/shared/lib/phone.ts')
						: require(name),
		})
		cache.set(file, exports)
		return exports
	}
	return {
		data,
		load,
		emit: (key) => events.forEach((fn) => fn({ key })),
		window,
		blockRead: (value) => {
			readFailure = value
		},
		blockWrite: (value) => {
			writeFailure = value
		},
		writes: () => writes,
		reload: () => cache.clear(),
	}
}
const fields = {
	recipientName: 'Иван',
	recipientPhone: '+79991234567',
	name: 'Дом',
	city: 'Белореченск',
	street: 'ул. Гоголя, 61',
	apartment: '12',
	entrance: '2, этаж 3',
}
const fixture = environment()
const { addressFieldsSchema } = fixture.load('src/entities/address/model/schema.ts')
const { formatAddress, addressDuplicateKey } = fixture.load('src/entities/address/model/format.ts')
const storePath = 'src/entities/address/model/guest-store.ts'
const json = (value) => JSON.parse(JSON.stringify(value))
function success(result) {
	assert.equal(result.ok, true, result.message)
	return result
}
function failure(result, message) {
	assert.equal(result.ok, false)
	assert.equal(result.message, message)
}
function entry(fields = {}) {
	return { ...json(defaultEntry), ...fields }
}
const defaultEntry = { ...fields, id: randomUUID(), revision: randomUUID(), isDefault: true }
function book(entries = [entry()]) {
	return { version: 1, revision: randomUUID(), entries }
}
function loadStore(env) {
	return env.load(storePath)
}
function read(env) {
	return JSON.parse(env.data.get(KEY))
}

// guestSchemaAndFormatting: rune counts, exact field messages, Go-compatible whitespace and NFC fixtures.
for (const [field, max, message] of [
	['name', 40, 'Не больше 40 символов'],
	['city', 80, 'Не больше 80 символов'],
	['street', 120, 'Слишком длинный адрес'],
	['apartment', 20, 'Не больше 20 символов'],
	['entrance', 20, 'Не больше 20 символов'],
]) {
	assert.equal(addressFieldsSchema.safeParse({ ...fields, [field]: '😀'.repeat(max) }).success, true)
	const result = addressFieldsSchema.safeParse({ ...fields, [field]: '😀'.repeat(max + 1) })
	assert.equal(result.success, false)
	assert.equal(result.error.issues[0].message, message)
}
for (const [field, value, message] of [
	['city', '', 'Выберите населённый пункт'],
	['street', '12', 'Укажите улицу и дом'],
]) {
	assert.equal(addressFieldsSchema.safeParse({ ...fields, [field]: value }).error.issues[0].message, message)
}
assert.equal(addressFieldsSchema.parse({ ...fields, name: '\u0085\u00a0Дом\u2003' }).name, 'Дом')
assert.equal(formatAddress(fields), 'ул. Гоголя, 61, кв. 12, подъезд 2, этаж 3')
assert.equal(formatAddress({ ...fields, apartment: '', entrance: '' }), fields.street)
assert.equal(formatAddress({ ...fields, apartment: '' }), `${fields.street}, подъезд ${fields.entrance}`)
assert.equal(addressDuplicateKey(' БЕЛОРЕЧЕНСК\u00a0', ' МАИ\u0306СКАЯ\u00a0 1 '), '["белореченск","майская 1"]')
assert.notEqual(addressDuplicateKey('Город', 'ул. 1'), addressDuplicateKey('Город', 'ул 1'))
assert.notEqual(addressDuplicateKey('Город', 'ул 1, кв. 1'), addressDuplicateKey('Город', 'ул 1, подъезд 1'))
console.log(
	'PASS guestSchemaAndFormatting: expected exact messages, code-point limits, Unicode fixtures and unchanged formatting'
)

const env = environment(),
	store = loadStore(env)
// guestHydration: deterministic server snapshot regardless of persisted client contents.
const server = store.getServerGuestSnapshot()
assert.equal(server.status, 'loading')
assert.strictEqual(server, store.getServerGuestSnapshot())
assert.equal(server.book.entries.length, 0)
assert.equal(store.getGuestSnapshot().status, 'ready')
assert.strictEqual(store.getGuestSnapshot(), store.getGuestSnapshot())
assert.strictEqual(store.getServerGuestSnapshot(), server)
// Evaluate without a browser global, as SSR does.
const ssrExports = {}
vm.runInNewContext(
	ts.transpileModule(readFileSync(storePath, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } })
		.outputText,
	{ exports: ssrExports, require: (name) => fixture.load(resolve(dirname(storePath), name + '.ts')) }
)
assert.strictEqual(ssrExports.getGuestSnapshot(), ssrExports.getServerGuestSnapshot())
console.log('PASS guestHydration: stable loading/empty server snapshot and stable client snapshots')

const first = success(await store.saveGuestAddress(fields)).entry
assert.equal(first.isDefault, true)
const revision = read(env).revision
const duplicate = success(await store.saveGuestAddress({ ...fields, name: 'Работа', city: ' БЕЛОРЕЧЕНСК ' }))
assert.equal(duplicate.created, false)
assert.equal(duplicate.entry.name, 'Дом')
assert.equal(read(env).revision, revision)
const second = success(await store.saveGuestAddress({ ...fields, street: 'Другой дом 2' })).entry
assert.equal(second.isDefault, false)
failure(await store.saveGuestAddress({ ...fields, street: second.street }, first.id), 'Такой адрес уже сохранён')
const renamed = success(await store.saveGuestAddress({ ...fields, name: 'Мой дом' }, first.id)).entry
assert.notEqual(renamed.revision, first.revision)
success(await store.setGuestDefault(second.id))
assert.equal(read(env).entries[0].isDefault, false)
assert.notEqual(read(env).entries[0].revision, renamed.revision)
assert.equal(read(env).entries[1].isDefault, true)
assert.notEqual(read(env).entries[1].revision, second.revision)
success(await store.deleteGuestAddress(second.id))
assert.equal(
	read(env).entries.some((e) => e.isDefault),
	false
)
assert.equal(success(await store.saveGuestAddress({ ...fields, street: 'Новый дом 3' })).entry.isDefault, false)
failure(await store.setGuestDefault('missing'), 'Адрес больше недоступен')
failure(await store.saveGuestAddress(fields, 'missing'), 'Адрес больше недоступен')
success(await store.deleteGuestAddress('missing'))
env.reload()
assert.deepEqual(json(loadStore(env).getGuestSnapshot().book), read(env))
console.log('PASS guestCrudAndDefault: duplicate preservation, edit/collision/default/delete/missing IDs and reload')

const badData = [
	'{',
	'{}',
	JSON.stringify({ ...book(), entries: [entry(), entry()] }),
	JSON.stringify(book([entry(), entry({ id: randomUUID() })])),
	JSON.stringify(book([entry({ city: 7 })])),
]
for (const raw of badData) {
	const env = environment(),
		s = loadStore(env)
	env.data.set(KEY, raw)
	assert.equal(s.getGuestSnapshot().status, 'corrupt')
	failure(await s.saveGuestAddress(fields), 'Не удалось прочитать сохранённые адреса')
	for (const operation of [
		() => s.deleteGuestAddress('x'),
		() => s.setGuestDefault('x'),
		() => s.removeImportedGuests([]),
	])
		assert.equal((await operation()).ok, false)
	assert.equal(env.data.get(KEY), raw)
	assert.equal(env.writes(), 0) // Cancel/reset not called: raw remains.
	success(await s.resetGuestAddresses())
	assert.deepEqual(read(env).entries, [])
}
for (const version of [2, 0, '1']) {
	const env = environment(),
		s = loadStore(env),
		raw = JSON.stringify({ version })
	env.data.set(KEY, raw)
	assert.equal(s.getGuestSnapshot().status, 'unsupported')
	failure(await s.saveGuestAddress(fields), 'Обновите страницу, чтобы открыть сохранённые адреса')
	assert.equal((await s.resetGuestAddresses()).ok, false)
	assert.equal(env.data.get(KEY), raw)
}
{
	const env = environment(),
		s = loadStore(env)
	env.blockRead(true)
	assert.equal(s.getGuestSnapshot().status, 'unavailable')
	failure(await s.saveGuestAddress(fields), 'Не удалось сохранить адрес в этом браузере')
	env.blockRead(false)
	assert.equal(s.getGuestSnapshot().status, 'ready')
	env.blockWrite(true)
	failure(await s.saveGuestAddress(fields), 'Не удалось сохранить адрес в этом браузере')
	assert.equal(env.data.has(KEY), false)
	assert.equal(s.getGuestSnapshot().book.entries.length, 0)
	assert.equal(fields.street, 'ул. Гоголя, 61')
	env.blockWrite(false)
	success(await s.saveGuestAddress(fields))
	const raw = env.data.get(KEY)
	env.blockWrite(true)
	failure(await s.deleteGuestAddress(read(env).entries[0].id), 'Не удалось сохранить адрес в этом браузере')
	assert.equal(env.data.get(KEY), raw)
	env.blockWrite(false)
	env.data.set(KEY, '{')
	env.blockWrite(true)
	assert.equal((await s.resetGuestAddresses()).ok, false)
	assert.equal(env.data.get(KEY), '{')
}
console.log(
	'PASS guestStorageFailures: corrupt/unsupported preserved, explicit reset, blocked read/write/quota and no false success'
)

{
	const env = environment(),
		s = loadStore(env)
	let notifications = 0
	const unsubscribe = s.subscribeToGuestAddresses(() => notifications++)
	const other = book()
	env.data.set(KEY, JSON.stringify(other))
	env.emit(KEY)
	assert.equal(notifications, 1)
	assert.equal(env.writes(), 0)
	assert.equal(s.getGuestSnapshot().book.revision, other.revision)
	const next = book([entry(), entry({ id: randomUUID(), street: 'Дом 2', isDefault: false })])
	env.data.set(KEY, JSON.stringify(next)) // No event yet: mutation must re-read anyway.
	success(await s.saveGuestAddress({ ...fields, street: 'Дом 3' }))
	assert.equal(read(env).entries.length, 3)
	env.data.delete(KEY)
	env.emit(null)
	assert.equal(s.getGuestSnapshot().book.entries.length, 0)
	const count = notifications
	unsubscribe()
	env.emit(KEY)
	assert.equal(notifications, count)
}
console.log('PASS guestCrossTab: update/clear notify without writes; mutations re-read sequential changes')

{
	const env = environment(),
		s = loadStore(env)
	const a = success(await s.saveGuestAddress(fields)).entry
	const b = success(await s.saveGuestAddress({ ...fields, street: 'Дом 2' })).entry
	const acknowledged = [a, b].map(({ id, revision }) => ({ id, revision }))
	const next = read(env)
	next.entries[1].name = 'Concurrent edit'
	next.entries[1].revision = randomUUID()
	next.revision = randomUUID()
	env.data.set(KEY, JSON.stringify(next))
	const c = success(await s.saveGuestAddress({ ...fields, street: 'Дом 3' })).entry
	success(await s.removeImportedGuests(acknowledged))
	assert.deepEqual(
		read(env).entries.map((e) => e.id),
		[b.id, c.id]
	)
	assert.equal(
		read(env).entries.some((e) => e.isDefault),
		false
	)
}
console.log(
	'PASS guestImportCleanup: exact id+revision acknowledgment, concurrent edits/new entries retained, no promotion'
)

{
	const env = environment(),
		s = loadStore(env)
	success(
		await s.saveGuestAddress({ ...fields, phone: 'private', session: 'secret', latitude: 12, account: { id: 1 } })
	)
	assert.deepEqual(Object.keys(read(env)).sort(), ['entries', 'revision', 'version'])
	assert.deepEqual(Object.keys(read(env).entries[0]).sort(), [
		'apartment',
		'city',
		'entrance',
		'id',
		'isDefault',
		'name',
		'recipientName',
		'recipientPhone',
		'revision',
		'street',
	])
	assert.equal(env.data.size, 1)
	assert.equal(/private|secret|latitude|account/.test(env.data.get(KEY)), false)
}
console.log('PASS guestPrivacy: only declared guest fields and the single versioned guest envelope serialized')

{
	const legacy = { ...fields }
	delete legacy.recipientName
	delete legacy.recipientPhone
	const env = environment()
	env.data.set(
		KEY,
		JSON.stringify({
			version: 1,
			revision: randomUUID(),
			entries: [{ ...legacy, id: randomUUID(), revision: randomUUID(), isDefault: true }],
		})
	)
	const store = loadStore(env)
	assert.equal(store.getGuestSnapshot().status, 'ready')
	assert.equal(store.getGuestSnapshot().book.entries[0].recipientName, '')
	assert.equal((await store.saveGuestAddress(legacy)).ok, false)
}
console.log('PASS legacy contacts: old guest entries load, new saves require contacts')
