/** Pure monitoring checks: no SDK initialization or outbound telemetry. */
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import ts from 'typescript'

const source = await readFile(new URL('../../src/shared/monitoring/config.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
	compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
})
const dataSource = await readFile(new URL('../../src/shared/monitoring/autofill-data.ts', import.meta.url), 'utf8')
const dataJS = ts.transpileModule(dataSource, {
	compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
}).outputText
const configJS = outputText.replace(
	"'./autofill-data'",
	JSON.stringify(`data:text/javascript;base64,${Buffer.from(dataJS).toString('base64')}`)
)
const { sampleRate, backendTraceTargets, scrubError, scrubSpan, privacyOptions } = await import(
	`data:text/javascript;base64,${Buffer.from(configJS).toString('base64')}`
)

assert.equal(sampleRate(undefined, true), 1)
assert.equal(sampleRate(undefined, false), 0.1)
for (const value of ['0', '1', '0.25']) assert.equal(sampleRate(value, false), Number(value))
for (const value of ['NaN', 'Infinity', '-0.1', '1.1', 'nope', ' ']) assert.throws(() => sampleRate(value, false))
assert.deepEqual(backendTraceTargets(undefined), [])
const [target] = backendTraceTargets('https://api.example.test:443/api/v1')
assert.ok(target.test('https://api.example.test/api/v1/products'))
for (const url of [
	'https://api.example.test.evil.test/',
	'https://evil.test/api.example.test',
	'http://api.example.test/',
	'https://apiXexampleXtest/',
])
	assert.equal(target.test(url), false, url)
assert.throws(() => backendTraceTargets('not a URL'))

const event = scrubError({
	type: undefined,
	event_id: 'safe',
	release: 'web@test',
	environment: 'test',
	request: { url: 'https://site.test/account?secret=secret', headers: { cookie: 'secret' }, data: 'secret' },
	message: 'secret',
	extra: { secret: 'secret' },
	user: { email: 'secret' },
	breadcrumbs: [{ message: 'secret' }],
	contexts: {
		private: { secret: 'secret' },
		trace: { trace_id: 'trace', span_id: 'span', data: { secret: 'secret' } },
	},
	exception: {
		values: [
			{
				type: 'Error',
				value: 'secret',
				mechanism: { type: 'generic', handled: false, data: { secret: 'secret' } },
				stacktrace: {
					frames: [{ filename: 'app.js?secret=secret', function: 'handler', lineno: 42, vars: { secret: 'secret' } }],
				},
			},
		],
	},
})
assert.equal(JSON.stringify(event).includes('secret'), false)
assert.equal(event.exception.values[0].stacktrace.frames[0].lineno, 42)
assert.equal(event.contexts.trace.trace_id, 'trace')
assert.equal(event.release, 'web@test')
assert.deepEqual(event.user, { ip_address: '0.0.0.0' })
assert.equal(scrubError({}).message, 'Unexpected application error')
const span = scrubSpan({
	trace_id: 'trace',
	span_id: 'span',
	name: 'secret',
	start_timestamp: 1,
	end_timestamp: 2,
	status: 'ok',
	is_segment: true,
	attributes: { 'sentry.op': 'http.client', 'http.url': 'secret', 'user.email': 'secret' },
	links: [{ attributes: { secret: 'secret' } }],
})
assert.equal(JSON.stringify(span).includes('secret'), false)
assert.equal(span.name, 'http.client')
assert.equal(span.trace_id, 'trace')
assert.equal(scrubSpan({ attributes: {}, status: 'ok' }).name, 'operation')
assert.equal(privacyOptions.beforeBreadcrumb({ message: 'secret' }), null)
assert.deepEqual(privacyOptions.dataCollection.httpBodies, [])

for (const file of ['instrumentation-client.ts', 'sentry.server.config.ts', 'sentry.edge.config.ts']) {
	const config = await readFile(new URL(`../../${file}`, import.meta.url), 'utf8')
	assert.match(config, /enabled: Boolean\(process\.env\./, `${file} must disable reporting without a DSN`)
	assert.doesNotMatch(config, /SENTRY_AUTH_TOKEN/, 'build token must never enter SDK initialization')
}
console.log('Sentry QA: sampling, propagation boundaries, error/span redaction and disabled configuration pass.')

const diagnostic = scrubError({
	tags: { autofill_reason: 'provider_http', secret: 'secret' },
	fingerprint: ['secret'],
	contexts: {
		autofill: { stage: 'provider', http_status: 403, elapsed_ms: 20, lat: 44, address: 'secret', surface: 'secret' },
	},
})
assert.deepEqual(diagnostic.tags, { autofill_reason: 'provider_http' })
assert.deepEqual(diagnostic.contexts.autofill, { stage: 'provider', http_status: 403, elapsed_ms: 20 })
assert.deepEqual(diagnostic.fingerprint, ['address-autofill', 'provider', 'provider_http'])
assert.equal(JSON.stringify(diagnostic).includes('secret'), false)
assert.equal(
	scrubError({ tags: { autofill_reason: 'customer_address' }, contexts: { autofill: { stage: 'provider' } } }).tags,
	undefined
)
console.log('Autofill reason/context/fingerprint allowlist PASS')
