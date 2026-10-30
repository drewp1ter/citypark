import { spawnSync } from 'node:child_process'

const required = ['BASE_URL', 'MENU_API_URL']
const missing = required.filter((name) => !process.env[name])
if (missing.length) {
	console.error(`FAIL: set ${missing.join(', ')} to the production-build website and running backend API URLs.`)
	process.exit(1)
}

const base = new URL(process.env.BASE_URL)
const api = new URL(process.env.MENU_API_URL)
if (base.protocol !== 'http:' && base.protocol !== 'https:') throw new Error('BASE_URL must be HTTP(S)')
if (api.protocol !== 'http:' && api.protocol !== 'https:') throw new Error('MENU_API_URL must be HTTP(S)')

const checks = [
	['customer auth', 'flows/customer-auth.mjs'],
	['customer recovery and Google', 'flows/customer-recovery-google.mjs'],
	['customer history', 'flows/customer-history.mjs'],
	['Telegram Mini App', 'flows/telegram-mini.mjs'],
	['Telegram pairing and payment return', 'flows/telegram-pairing-payment.mjs'],
	['menu', 'flows/menu.mjs'],
	['checkout and contacts', 'flows/checkout-and-contacts.mjs'],
	['keyboard', 'keyboard.mjs'],
	['accessibility', 'a11y.mjs'],
	['responsive', 'responsive.mjs'],
]

async function reachable(url, label) {
	try {
		const response = await fetch(url, { signal: AbortSignal.timeout(10_000), redirect: 'manual' })
		console.log(`${label}: expected reachable; actual HTTP ${response.status}`)
		return response.ok
	} catch (error) {
		console.error(`${label}: expected reachable; actual ${error.cause?.code ?? error.name}. Start the configured service and retry.`)
		return false
	}
}

const ready = (await Promise.all([
	reachable(base, 'Production website'),
	reachable(new URL('categories', api.href.replace(/\/?$/, '/')), 'Backend API'),
])).every(Boolean)
if (!ready) process.exit(1)

if (process.env.QA_PLACE_ORDER === '1' && process.env.QA_CUSTOMER_WRITES !== '1') {
	console.error('FAIL: QA_PLACE_ORDER=1 requires owner approval and QA_CUSTOMER_WRITES=1.')
	process.exit(1)
}
if (process.env.QA_CUSTOMER_WRITES === '1' && process.env.QA_CUSTOMER_RELEASE_APPROVED !== '1') {
	console.error('FAIL: approved live checks require QA_CUSTOMER_RELEASE_APPROVED=1 as well as QA_CUSTOMER_WRITES=1.')
	process.exit(1)
}
if (process.env.QA_CUSTOMER_WRITES !== '1') {
	console.log('PENDING: registration, email/recovery, Google, pairing, order, payment and provider/device checks need owner approval and QA_CUSTOMER_WRITES=1; no backend writes will run.')
}

let failures = 0
for (const [label, file] of checks) {
	console.log(`\n== ${label}: expected PASS ==`)
	const result = spawnSync(process.execPath, [new URL(file, import.meta.url).pathname], {
		stdio: 'inherit',
		env: process.env,
	})
	const actual = result.error ? result.error.code ?? result.error.name : `exit ${result.status ?? `signal ${result.signal}`}`
	const passed = !result.error && result.status === 0
	console.log(`${label}: expected PASS; actual ${passed ? 'PASS' : `FAIL (${actual})`}`)
	if (!passed) failures++
}
console.log(`\nAutomated checks: expected ${checks.length} PASS; actual ${checks.length - failures} PASS, ${failures} FAIL.`)
console.log('Live provider and Android/iOS/Desktop checks: PENDING until recorded in docs/CUSTOMER_ACCOUNTS_RELEASE.md.')
process.exitCode = failures ? 1 : 0
