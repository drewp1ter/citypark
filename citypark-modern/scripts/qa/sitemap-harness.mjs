import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
let rooms = [], news = [], failure, newsFailure
const exports = {}
const source = ts.transpileModule(readFileSync('src/app/search/sitemap.ts', 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
vm.runInNewContext(source, { exports, require: name => ({
	'@/entities/room': { getRooms: async () => { if (failure) throw failure; return rooms } },
	'@/entities/article': { getNews: async () => { if (newsFailure) throw newsFailure; return news } },
	'@/shared/config': { productionOrigin: 'https://cityparkvip.ru' },
})[name] })
const urls = async () => Array.from(await exports.default(), entry => entry.url)
assert.equal((await urls()).length, 10)
rooms = [{ id: 'vip' }, { id: 'vip' }]
news = [{ slug: 'новости ?&' }]
const result = await urls()
assert.equal(result.length, 12)
assert.ok(result.includes('https://cityparkvip.ru/rooms/vip'))
assert.ok(result.includes('https://cityparkvip.ru/news/' + encodeURIComponent(news[0].slug)))
assert.ok(result.every(url => !new URL(url).search && !/\/(account|checkout|mini|api)(\/|$)/.test(new URL(url).pathname)))
failure = Error('backend unavailable')
await assert.rejects(exports.default(), /backend unavailable/)
failure = undefined
newsFailure = Error('news unavailable')
await assert.rejects(exports.default(), /news unavailable/)
console.log('PASS: public routes, deduplication, encoded identifiers, private exclusions and failure propagation')
