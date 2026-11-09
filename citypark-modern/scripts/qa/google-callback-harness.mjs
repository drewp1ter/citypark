import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'

let proof, result, session, saved, failure
const modules = {
 'server-only': {},
 'next/navigation': {},
 '@/shared/customer': {
  consumeGoogleAttempt: async () => proof,
  getCustomerSessionToken: async () => session,
  customerRequest: async () => { if (failure) throw Error('Unavailable'); return result },
  setCustomerSession: async (...args) => { saved = args },
 },
}
const exports = {}
const source = ts.transpileModule(readFileSync('src/pages/account/api/google.ts', 'utf8'), {
 compilerOptions: { module: ts.ModuleKind.CommonJS },
}).outputText
vm.runInNewContext(source, { exports, require: name => modules[name], URL, Response })
const state = 'a'.repeat(32)
for (const origin of ['http://0.0.0.0:3000', 'http://localhost:3111', 'https://modern.cityparkvip.ru']) {
 for (const scenario of ['login', 'link', 'expired', 'missing', 'cancelled', 'failure', 'invalid']) {
  proof = scenario === 'missing' ? undefined : { intent: ['link', 'expired'].includes(scenario) ? 'link' : 'login', token: 'proof' }
  result = scenario === 'link' ? { linked: true } : { token: 'session', expiresAt: 'expiry' }
  session = scenario === 'expired' ? undefined : 'existing'
  failure = ['cancelled', 'failure'].includes(scenario)
  saved = undefined
  const query = scenario === 'invalid' ? 'state=invalid&code=x' : 'state=' + state + (scenario === 'cancelled' ? '&error=access_denied' : '&code=x')
  const response = await exports.completeGoogleCallback(new Request(origin + '/account/google/callback?' + query))
  const expected = scenario === 'login' ? '/menu' : scenario === 'link' ? '/account?linked=google' : '/account/sign-in?google=' + (scenario === 'expired' ? 'expired' : scenario === 'cancelled' ? 'cancelled' : 'failed')
  assert.equal(response.status, 303)
  assert.equal(response.headers.get('location'), expected)
  assert.equal(new URL(response.headers.get('location'), 'https://modern.cityparkvip.ru/account/google/callback').origin, 'https://modern.cityparkvip.ru')
  assert.equal(response.headers.get('cache-control'), 'private, no-store')
  assert.equal(response.headers.get('referrer-policy'), 'no-referrer')
  assert.equal(Boolean(saved), scenario === 'login')
 }
}
console.log('PASS: Google callback redirects retain public origin across login, link, and error outcomes')
