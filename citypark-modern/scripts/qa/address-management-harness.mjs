// Isolated Server Action and entity checks. No live backend mutations.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import vm from 'node:vm'
import ts from 'typescript'
const require = createRequire(import.meta.url)
const compile = (path) =>
	ts.transpileModule(readFileSync(path, 'utf8'), {
		compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS },
	}).outputText
const phones = {}
vm.runInNewContext(compile('src/shared/lib/phone.ts'), { exports: phones, require })
const schemas = {}
vm.runInNewContext(compile('src/entities/address/model/schema.ts'), {
	exports: schemas,
	require: (name) => (name === '@/shared/lib' ? phones : require(name)),
})
class CustomerApiError extends Error {
	constructor(status, message, fields = {}) {
		super(message)
		this.status = status
		this.fields = fields
	}
}
const calls = [],
	paths = []
let failure
const entry = {
	id: 1,
	recipientName: 'Иван',
	recipientPhone: '+79991234567',
	name: 'Дом',
	city: 'Белореченск',
	street: 'Гоголя, 61',
	apartment: '',
	entrance: '',
	isDefault: true,
	needsReview: false,
}
const fields = {
	recipientName: entry.recipientName,
	recipientPhone: entry.recipientPhone,
	name: entry.name,
	city: entry.city,
	street: entry.street,
	apartment: '',
	entrance: '',
}
let created = true
const helpers = Object.fromEntries(
	['createSavedAddress', 'updateSavedAddress', 'deleteSavedAddress', 'setSavedDefault'].map((name) => [
		name,
		async (...args) => {
			calls.push({ name, args })
			if (failure) throw failure
			return name === 'createSavedAddress' ? { entry, created } : entry
		},
	])
)
const exports = {}
vm.runInNewContext(compile('src/features/manage-addresses/api/actions.ts'), {
	exports,
	require: (name) => {
		if (name === 'next/cache') return { revalidatePath: (path) => paths.push(path) }
		if (name === '@/entities/address') return schemas
		if (name === '@/entities/address/index.server') return helpers
		if (name === '@/shared/customer') return { CustomerApiError }
		throw Error(name)
	},
})
for (const action of Object.values(exports)) {
	for (const input of [
		null,
		{},
		{ surface: 'invalid', expectedAccountId: 1 },
		{ surface: 'web', expectedAccountId: 0 },
		{ surface: 'mini', expectedAccountId: 1, id: -1 },
	]) {
		assert.equal((await action(input, fields)).ok, false)
	}
}
assert.equal(calls.length, 0)
const input = { surface: 'web', expectedAccountId: 7 }
assert.equal((await exports.saveAddressAction(input, { ...fields, street: '' })).fields.street, 'Укажите улицу и дом')
for (const surface of ['web', 'mini']) {
	const root = surface === 'web' ? '/account' : '/mini'
	for (const [action, id, helper] of [
		['saveAddressAction', undefined, 'createSavedAddress'],
		['saveAddressAction', 1, 'updateSavedAddress'],
		['deleteAddressAction', 1, 'deleteSavedAddress'],
		['defaultAddressAction', 1, 'setSavedDefault'],
	]) {
		calls.length = 0
		paths.length = 0
		const value = await exports[action]({ surface, expectedAccountId: 7, id }, fields)
		assert.equal(value.ok, true)
		assert.equal(calls[0].name, helper)
		assert.equal(calls[0].args[0], surface)
		assert.equal(calls[0].args[1], 7)
		assert.deepEqual(paths, [root + '/addresses', surface === 'web' ? '/checkout' : '/mini/checkout'])
		for (const [status, message] of [
			[409, 'Такой адрес уже сохранён'],
			[404, 'Не найдено'],
			[401, 'Войдите повторно, чтобы сохранить адрес'],
			[502, 'Не удалось сохранить адрес. Попробуйте ещё раз.'],
		]) {
			failure = new CustomerApiError(status, message)
			paths.length = 0
			const result = await exports[action]({ surface, expectedAccountId: 7, id }, fields)
			assert.equal(result.ok, false)
			assert.equal(result.message, message)
			assert.equal(result.authRequired, status === 401)
			assert.equal(paths.length, 0)
		}
		failure = new Error('private transport data')
		assert.equal(
			(await exports[action]({ surface, expectedAccountId: 7, id }, fields)).message,
			'Не удалось сохранить адрес. Попробуйте ещё раз.'
		)
		failure = undefined
	}
}
created = false
assert.equal((await exports.saveAddressAction(input, fields)).created, false)
await import('./address-api-harness.mjs')
await import('./address-store-harness.mjs')
console.log(
	'PASS managementActions: input validation, both surfaces, create/duplicate/edit/default/delete, conflict/missing/auth/transport, revalidation; entity ownership/session and guest-store checks'
)
