// Isolated failure/crypto coverage; does not replace authenticated backend acceptance.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import * as crypto from 'node:crypto'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
function load(file, modules) {
 const source = ts.transpileModule(readFileSync(file, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText
 const exports = {}
 vm.runInNewContext(source, { exports, require: name => { if (name in modules) return modules[name]; if (name === 'react/jsx-runtime') return require(name); throw Error(name) }, Buffer, Date, process: { env: { NODE_ENV: 'production' } }, URL, console, queueMicrotask })
 return exports
}
const jar = new Map()
const session = load('src/shared/customer/customer-session.ts', {
 'server-only': {}, 'node:crypto': crypto, '@/shared/config': { serverEnv: () => ({ CUSTOMER_PUBLIC_ORIGIN: 'https://example.com' }) },
 'next/headers': { cookies: async () => ({ get: name => jar.has(name) ? { value: jar.get(name) } : undefined, set: (name, value) => jar.set(name, value) }) }
})
const transactionId = '12345678-abcd-abcd-abcd-123456789abc'
const paymentUrl = `https://demo.moneta.ru/assistant.htm?MNT_TRANSACTION_ID=${transactionId}`
const fallback = 'Не удалось перейти к оплате. Попробуйте ещё раз.'
class ApiError extends Error { constructor(status, message) { super(message); this.status = status } }
const redirect = path => { throw Object.assign(Error('redirect'), { path }) }
let web = 'web-token', mini = 'mini-token', calls, result, failure, signed, refreshed
const action = load('src/features/payment-retry/api/retry-payment.ts', {
 'next/navigation': { redirect }, 'next/cache': { revalidatePath: path => refreshed.push(path) },
 '@/shared/customer': { CustomerApiError: ApiError, transactionIdFromRedirect: session.transactionIdFromRedirect, getCustomerSessionToken: async () => web, getMiniSessionToken: async () => mini,
 customerRequest: async (path, init) => { calls.push({ path, init }); if (failure) throw failure; return result },
 setPendingPaymentContext: async (...args) => { signed = args } }
}).retryPayment
function reset() { calls = []; signed = null; refreshed = []; failure = null; result = { orderId: 7, redirectUrl: paymentUrl }; web = 'web-token'; mini = 'mini-token' }
for (const surface of ['web', 'mini']) {
 reset()
 await assert.rejects(action(7, surface, {}, new FormData()), e => e.path === result.redirectUrl)
 assert.equal(calls[0].path, '/orders/7/payment-retry'); assert.equal(calls[0].init.method, 'POST'); assert.equal(calls[0].init.body, undefined)
 assert.equal(calls[0].init.headers.Authorization, `Bearer ${surface}-token`)
 assert.equal(JSON.stringify(signed), JSON.stringify([7, transactionId, surface, 'history']))
 for (const status of [401, 404, 409, 500, 503]) {
  reset(); failure = new ApiError(status, 'denied')
  if (status === 401) await assert.rejects(action(7, surface, {}, new FormData()), e => e.path === (surface === 'mini' ? '/mini' : '/account/sign-in?returnTo=%2Faccount%2Forders%2F7'))
  else { const state = await action(7, surface, {}, new FormData()); assert.equal(state.error, status === 404 ? 'Заказ недоступен.' : status === 409 ? 'denied' : fallback); assert.equal(refreshed.length, status === 409 ? 2 : 0) }
 }
 reset(); failure = new ApiError(503, 'Оплата доступна только в часы работы ресторана.'); assert.equal((await action(7, surface, {}, new FormData())).error, failure.message); assert.equal(refreshed.length, 2)
 reset(); if (surface === 'mini') mini = undefined; else web = undefined
 await assert.rejects(action(7, surface, {}, new FormData()), e => !!e.path); assert.equal(calls.length, 0)
}
for (const bad of [0, -1, 1.5, NaN, '7', Number.MAX_SAFE_INTEGER + 1]) { reset(); assert.equal((await action(bad, 'web', {}, new FormData())).error, fallback); assert.equal(calls.length, 0) }
reset(); assert.equal((await action(7, 'guest', {}, new FormData())).error, fallback); assert.equal(calls.length, 0)
for (const malformed of [{}, { orderId: 8, redirectUrl: 'https://demo.moneta.ru' }, { orderId: 7, redirectUrl: '' }, { orderId: 7, redirectUrl: 'javascript:alert(1)' }]) { reset(); result = malformed; assert.equal((await action(7, 'web', {}, new FormData())).error, fallback); assert.equal(signed, null) }
reset(); failure = Error('network'); assert.equal((await action(7, 'web', {}, new FormData())).error, fallback)
console.log('PASS: action surface isolation, saved ID/no amount, success redirects, auth/denials, refresh, malformed/network failures')

jar.set('__Host-citypark_customer', 'web-token'); jar.set('__Host-citypark_mini', 'mini-token')
function cookie(context, key = 'web-token') { const payload = Buffer.from(JSON.stringify(context)).toString('base64url'); jar.set('__Host-citypark_payment', `${payload}.${crypto.createHmac('sha256', key).update(payload).digest('base64url')}`) }
for (const surface of ['web', 'mini']) for (const origin of ['checkout', 'history']) {
 await session.setPendingPaymentContext(7, transactionId, surface, origin)
 assert.equal((await session.getPendingPaymentContext(transactionId)).origin, origin)
 assert.equal(await session.getPendingPaymentContext('00000000-0000-0000-0000-000000000000'), undefined)
 const raw = jar.get('__Host-citypark_payment'); jar.set('__Host-citypark_payment', raw + 'x'); assert.equal(await session.getPendingPaymentContext(transactionId), undefined)
}
cookie({ orderId: 7, transactionId, surface: 'web', issuedAt: Date.now() }); assert.equal((await session.getPendingPaymentContext(transactionId)).origin, 'checkout')
for (const patch of [{ origin: 'evil' }, { issuedAt: Date.now() - 86400_001 }, { issuedAt: Date.now() + 60000 }, { surface: 'guest' }]) { cookie({ orderId: 7, transactionId, surface: 'web', issuedAt: Date.now(), ...patch }); assert.equal(await session.getPendingPaymentContext(transactionId), undefined) }
cookie({ orderId: 7, transactionId, surface: 'web', issuedAt: Date.now(), origin: 'history' }, 'other-session'); assert.equal(await session.getPendingPaymentContext(transactionId), undefined)
console.log('PASS: signed origins, old cookies, tampering, expiry, transaction and session binding')

const { renderToStaticMarkup } = require('react-dom/server')
const uiModules = { react: { useEffect() {}, useActionState: () => [{}, () => {}, false], useRef: () => ({ current: false }) }, '@/shared/ui': { Button: props => require('react').createElement('button', props) }, '../api/retry-payment': { retryPayment: () => {} } }
const { PaymentRetry } = load('src/features/payment-retry/ui/payment-retry.tsx', uiModules)
for (const surface of ['web', 'mini']) for (const status of ['new', 'pending', 'paid', 'confirmed', 'canceled']) for (const eligibility of [undefined, { allowed: true, reason: '' }, { allowed: false, reason: 'expired' }, { allowed: false, reason: 'closed' }, { allowed: false, reason: 'not_payable' }, { allowed: true, reason: 'unknown' }]) {
 const html = renderToStaticMarkup(require('react').createElement(PaymentRetry, { orderId: 7, surface, status, eligibility }))
 const unpaid = ['new', 'pending'].includes(status)
 assert.equal(html.includes('Оплатить'), unpaid && eligibility?.allowed === true && eligibility.reason === '')
 assert.equal(html.includes('Срок оплаты'), unpaid && eligibility?.allowed === false && eligibility.reason === 'expired')
 assert.equal(html.includes('часы работы'), unpaid && eligibility?.allowed === false && eligibility.reason === 'closed')
}
const { PaymentReturnStatus } = load('src/features/payment-retry/ui/payment-return-status.tsx', { 'next/navigation': { useRouter: () => ({ refresh() {} }) }, '@/shared/ui': uiModules['@/shared/ui'] })
for (const status of ['paid', 'confirmed', 'pending', 'canceled']) for (const payment of ['returned', 'failed', undefined]) {
 const html = renderToStaticMarkup(require('react').createElement(PaymentReturnStatus, { status, payment }))
 assert.equal(html.includes('Заказ оплачен'), ['paid', 'confirmed'].includes(status))
 assert.equal(html.includes('Обновить статус'), status === 'pending' && payment === 'returned')
 assert.equal(html.includes('Оплата не завершена'), status === 'pending' && payment === 'failed')
}
console.log('PASS: both surfaces eligibility/status matrix, unknown metadata, confirmed payment beats failure query')
for (const [file, name, props, failed] of [
 ['src/pages/checkout/ui/checkout-done-page.tsx', 'CheckoutDonePage', { searchParams: Promise.resolve({ MNT_TRANSACTION_ID: transactionId }) }, false],
 ['src/pages/checkout/ui/checkout-page.tsx', 'CheckoutPage', { paymentFailed: true, returnedOrderId: transactionId }, true],
 ['src/pages/mini/ui/mini-payment-return-page.tsx', 'MiniPaymentReturnPage', { searchParams: Promise.resolve({ MNT_TRANSACTION_ID: transactionId }) }, false],
 ['src/pages/mini/ui/mini-payment-return-page.tsx', 'MiniPaymentReturnPage', { searchParams: Promise.resolve({ MNT_TRANSACTION_ID: transactionId }), failed: true }, true]
]) for (const surface of ['web', 'mini']) {
 const modules = {
 '@/entities/address/index.server': {}, 'next/navigation': { redirect }, 'next/server': {}, '@/shared/customer': { getPendingPaymentContext: async () => ({ orderId: 7, transactionId, surface, origin: 'history' }) },
 '@/entities/order': { getMyOrder: () => { throw Error('Must route history before checkout effects') } }, '@/shared/ui': {}, '@/shared/api': {}, '@/shared/lib': {}, '@/shared/config': {}, '@/entities/dish': {}, '@/widgets/food-ordering': {},
 './checkout-header': {}, './clear-cart': {}, './title-row': {}, './mini-clear-cart': {}, './pending-payment-refresh': {}
 }
 await assert.rejects(load(file, modules)[name](props), e => e.path === `/${surface === 'mini' ? 'mini' : 'account'}/orders/7?payment=${failed ? 'failed' : 'returned'}`)
}
console.log('PASS: both surfaces route signed history success/failure before checkout reads/cart effects')

for (const surface of ['web', 'mini']) {
 for (const id of [undefined, '7', 'invalid']) {
  reset(); result.redirectUrl = `https://demo.moneta.ru/assistant.htm${id ? '?MNT_TRANSACTION_ID=' + id : ''}`
  await assert.rejects(action(7, surface, {}, new FormData()), e => e.path === result.redirectUrl)
  assert.equal(signed, null)
 }
}
for (const id of [transactionId, transactionId.toUpperCase()]) {
 assert.equal(session.transactionIdFromRedirect(`https://demo.moneta.ru/?MNT_TRANSACTION_ID=${id}`), transactionId)
 await session.setPendingPaymentContext(7, transactionId, 'web')
 assert.equal((await session.getPendingPaymentContext(id)).transactionId, transactionId)
}
for (const url of ['not a url', 'https://demo.moneta.ru/', 'https://demo.moneta.ru/?MNT_TRANSACTION_ID=7']) assert.equal(session.transactionIdFromRedirect(url), undefined)
assert.equal(await session.getPendingPaymentContext('7'), undefined)
cookie({ orderId: 7, surface: 'web', issuedAt: Date.now() })
assert.equal(await session.getPendingPaymentContext(transactionId), undefined)
await assert.rejects(session.setPendingPaymentContext(7, 'not-a-uuid', 'web'), /Invalid transaction id/)
for (const failed of [false, true]) {
 const file = `src/pages/checkout/ui/checkout${failed ? '' : '-done'}-page.tsx`
 const name = failed ? 'CheckoutPage' : 'CheckoutDonePage'
 const props = failed ? { paymentFailed: true, returnedOrderId: transactionId } : { searchParams: Promise.resolve({ MNT_TRANSACTION_ID: transactionId }) }
 const modules = {
  '@/entities/address/index.server': {}, 'next/navigation': { redirect }, 'next/server': {},
  '@/shared/customer': { getPendingPaymentContext: async id => { assert.equal(id, transactionId); return { orderId: 7, transactionId, surface: 'mini', origin: 'checkout' } } },
  '@/entities/order': {}, '@/shared/ui': {}, '@/shared/api': {}, '@/shared/lib': {}, '@/shared/config': {}, '@/entities/dish': {}, '@/widgets/food-ordering': {},
  './checkout-header': {}, './clear-cart': {}, './title-row': {}
 }
 await assert.rejects(load(file, modules)[name](props), e => e.path === `/mini/checkout${failed ? '?payment=failed&' : '/done?'}MNT_TRANSACTION_ID=${transactionId}`)
}
console.log('PASS: UUID parsing/matching, legacy rejection, missing-ID payment handoff, and Mini checkout success/failure forwarding')
