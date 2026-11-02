/** Controlled provider responses exercise the production parser without contacting DaData. */
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import ts from 'typescript'
const url = (code) =>
	`data:text/javascript;base64,${Buffer.from(ts.transpileModule(code, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }).outputText).toString('base64')}`
const error = await readFile(new URL('../../src/shared/api/geocode-error.ts', import.meta.url), 'utf8')
const source = (await readFile(new URL('../../src/shared/api/dadata.ts', import.meta.url), 'utf8'))
	.replace("import { GeocodeError } from './geocode-error'", `import { GeocodeError } from '${url(error)}'`)
	.replace(
		"import { serverEnv } from '@/shared/config'",
		"const serverEnv = () => ({ DADATA_API_KEY: 'synthetic-key' })"
	)
	.replace("import { dadataDebug } from './dadata-debug'", 'const dadataDebug = () => {}')
const { reverseGeocode } = await import(url(source))
const original = globalThis.fetch
try {
	for (const [mock, reason, status] of [
		[
			() => {
				throw new TypeError('private')
			},
			'provider_network',
		],
		[
			() => {
				throw new DOMException('private', 'TimeoutError')
			},
			'provider_timeout',
		],
		[() => new Response('private', { status: 403 }), 'provider_http', 403],
		[() => new Response('{'), 'provider_json', 200],
		[() => Response.json({}), 'provider_shape', 200],
		[() => Response.json({ suggestions: [null] }), 'provider_shape', 200],
		[() => Response.json({ suggestions: [{ data: { house: 1 } }] }), 'provider_shape', 200],
	]) {
		globalThis.fetch = async () => mock()
		await assert.rejects(
			reverseGeocode(44, 39),
			(error) => error.reason === reason && error.httpStatus === status && !error.message.includes('private')
		)
	}
	globalThis.fetch = async () => Response.json({ suggestions: [] })
	assert.equal(await reverseGeocode(44, 39), null)
	globalThis.fetch = async () =>
		Response.json({ suggestions: [{ data: { city: 'Synthetic', street_with_type: 'Synthetic street' } }] })
	assert.equal((await reverseGeocode(44, 39)).city, 'Synthetic')
	console.log('Provider network/timeout/HTTP/JSON/schema/empty/success cases PASS')
} finally {
	globalThis.fetch = original
}
