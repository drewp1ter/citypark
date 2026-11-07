import assert from 'node:assert/strict'
import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'

// Exercise the real adapter with isolated configuration and provider responses.
const dir = await mkdtemp(join(tmpdir(), 'address-regions-'))
const originalFetch = globalThis.fetch
try {
	let source = await readFile(new URL('../../src/shared/api/dadata.ts', import.meta.url), 'utf8')
	source = source.replace(/^import .*\n/gm, '')
	source =
		"const serverEnv = () => ({ DADATA_API_KEY: 'test-key' }); const dadataDebug = () => {}; class GeocodeError extends Error {}\n" +
		source
	const file = join(dir, 'dadata.ts')
	await writeFile(file, source)
	const { suggestAddress } = await import(pathToFileURL(file).href)
	for (const [town, code, region, name] of [
		['Белореченск', 'RU-KDA', 'Краснодарский', 'Белореченск'],
		['Майкоп', 'RU-AD', 'Адыгея', 'Майкоп'],
		['п. Гончарка', 'RU-AD', 'Адыгея', 'Гончарка'],
	]) {
		let payload
		globalThis.fetch = async (_url, init) => {
			payload = JSON.parse(init.body)
			return Response.json({
				suggestions: [{ value: 'full address', data: { street_with_type: 'ул Мира', house: '1', house_type: 'д' } }],
			})
		}
		assert.deepEqual(await suggestAddress('мира', town, code), ['ул Мира, д 1'])
		assert.deepEqual(payload.locations, [
			{ region, city: name },
			{ region, settlement: name },
		])
	}
	console.log('Region-specific suggestion payloads and settlement prefixes: PASS')
} finally {
	globalThis.fetch = originalFetch
	await rm(dir, { recursive: true, force: true })
}
