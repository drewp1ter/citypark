// Isolated menu integration regression; no requests to external services.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import { z } from 'zod'

function load(file, modules = {}, globals = {}) {
	const source = ts.transpileModule(readFileSync(file, 'utf8'), {
		compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
	}).outputText
	const exports = {}
	vm.runInNewContext(source, {
		exports,
		require: (name) => {
			assert.ok(name in modules, `Unexpected import: ${name}`)
			return modules[name]
		},
		URL,
		process: { env: {} },
		...globals,
	})
	return exports
}

const categories = [{ id: 1, name: 'Кухня', description: '', images: [], isAvailableForOrder: true }]
const product = (id, extra = {}) => ({
	id,
	categoryId: 1,
	name: `Блюдо ${id}`,
	description: '',
	weight: 0.2,
	price: 100,
	images: [],
	isHidden: false,
	isOrderBlocked: false,
	isAvailableForOrder: true,
	...extra,
})
const products = Array.from({ length: 501 }, (_, i) => product(i + 1, { isHidden: true }))
products[0] = product(1)
products[1] = product(2, { isHidden: true, isOrderBlocked: true })
products[2] = product(3, { categoryId: 2 })
products[3] = product(4, { isOrderBlocked: true, isAvailableForOrder: false })
products[500] = product(501)
delete products[500].isHidden // Older backend responses remain readable during rollout.
const requests = []
const api = load(
	'src/entities/dish/api/menu-api.ts',
	{ '@/shared/config': { serverEnv: () => ({ MENU_API_URL: 'https://menu.test' }) } },
	{
		fetch: async (url, init) => {
			assert.equal(init.cache, 'no-store')
			const parsed = new URL(url)
			requests.push(parsed.pathname + parsed.search)
			if (parsed.pathname === '/categories') return { ok: true, json: async () => categories }
			assert.equal(parsed.pathname, '/products')
			const page = Number(parsed.searchParams.get('page'))
			const size = Number(parsed.searchParams.get('pageSize'))
			return { ok: true, json: async () => products.slice((page - 1) * size, page * size) }
		},
	}
)
const mapping = load('src/entities/dish/api/to-dish.ts', {
	'@/shared/lib': { PLACEHOLDER_PHOTO: '/images/placeholder.svg', typo: (value) => value },
})
const { getMenu, getDishesByIds } = load('src/entities/dish/api/get-menu.ts', {
	react: { cache: (fn) => fn },
	'./menu-api': api,
	'./to-dish': mapping,
})
const { selectHomepageMenu } = load('src/entities/dish/lib/select-homepage-menu.ts')
const { priceLines } = load('src/entities/dish/lib/price-lines.ts')
const ids = (dishes) => Array.from(dishes, (dish) => dish.id)

let menu = await getMenu()
assert.deepEqual(ids(menu.dishes), ['1', '4', '501'])
assert.ok(requests.includes('/products?pageSize=500&page=2'), 'Filtering must not truncate pagination')
assert.deepEqual(ids(await getDishesByIds(['1', '2', '3'])), ['1'])
assert.equal(
	menu.dishes.find((dish) => dish.id === '2'),
	undefined,
	'Hidden dish routes resolve as missing'
)
const groups = selectHomepageMenu(
	menu,
	{ categories: [{ categoryId: 1, productIds: [2, 1] }] },
	'/images/placeholder.svg'
)
assert.deepEqual(ids(groups[0].dishes), ['1'], 'Saved hidden homepage IDs are omitted')
const cart = priceLines(
	[1, 2, 3, 4, 501].map((id) => ({ dishId: String(id), variantId: 'default', quantity: 2 })),
	menu.dishes
)
assert.deepEqual(ids(cart.items.map((line) => line.dish)), ['1', '501'])
assert.deepEqual(
	Array.from(cart.unavailable, (line) => [line.dishId, line.name]),
	[
		['2', undefined],
		['3', undefined],
		['4', 'Блюдо 4'],
	]
)
assert.equal(cart.subtotal, 400)
assert.equal(cart.count, 4)

const line = (id, variantId = 'default') => ({ dishId: String(id), variantId, quantity: 1 })
const closed = mapping.toDish(product(800, { isAvailableForOrder: false }))
const unpriced = mapping.toDish(product(801, { price: 0 }))
const staleLines = [line(1, 'old-size'), line(800), line(801)]
const staleCart = priceLines(staleLines, [...menu.dishes, closed, unpriced])
assert.equal(staleCart.subtotal, 0)
assert.deepEqual(
	Array.from(staleCart.unavailable, (line) => line.name),
	['Блюдо 1', 'Блюдо 800', 'Блюдо 801']
)
const restored = priceLines(staleLines, [...menu.dishes, mapping.toDish(product(800)), unpriced])
assert.deepEqual(ids(restored.items.map((line) => line.dish)), ['800'])
assert.equal(restored.unavailable.length, 2)
const serverRejected = priceLines([line(1), line(501)], menu.dishes, [line(1)])
assert.deepEqual(ids(serverRejected.items.map((line) => line.dish)), ['501'])
assert.equal(serverRejected.unavailable[0].name, 'Блюдо 1')

const zod = { z }
const schema = load('src/widgets/food-ordering/model/schema.ts', {
	zod,
	'@/shared/lib': { isValidRuPhone: () => true },
})
const quoteCalls = []
const { quoteOrder } = load('src/widgets/food-ordering/api/quote-order.ts', {
	zod,
	'@/entities/dish': { getMenu, priceLines },
	'../model/schema': schema,
	'./orders-api': { fetchOrderQuote: async (body) => quoteCalls.push(body) && { ok: true, quote: { subtotal: 100 } } },
})
const request = { lines: [line(1), line(2)], method: 'pickup', city: '', promoCode: '' }
const refusedQuote = await quoteOrder(request)
assert.equal(refusedQuote.ok, false)
assert.equal(JSON.stringify(refusedQuote.unavailableLines), JSON.stringify([{ dishId: '2', variantId: 'default' }]))
assert.equal(quoteCalls.length, 0)
const validQuote = await quoteOrder({ ...request, lines: [line(1)] })
assert.equal(validQuote.ok, true)
assert.equal(JSON.stringify(quoteCalls[0].items), JSON.stringify([{ productId: 1, amount: 1 }]))

// Drive the real quote hook with controlled requests and effect cleanup.
const cells = []
const effects = []
const requestsInFlight = []
let cursor = 0
const react = {
	useState(initial) {
		const i = cursor++
		if (!(i in cells)) cells[i] = initial
		return [
			cells[i],
			(value) => {
				cells[i] = value
			},
		]
	},
	useCallback: (fn) => fn,
	startTransition: (fn) => fn(),
	useEffect(fn, deps) {
		const i = cursor++
		const previous = cells[i]
		if (previous && deps.every((value, j) => value === previous.deps[j])) return
		effects.push(() => {
			previous?.cleanup?.()
			cells[i] = { deps, cleanup: fn() }
		})
	},
}
const { useCheckoutQuote: runCheckoutQuote } = load(
	'src/widgets/food-ordering/model/use-checkout-quote.ts',
	{
		react,
		'../api/quote-order': { quoteOrder: () => new Promise((resolve) => requestsInFlight.push(resolve)) },
	},
	{ sessionStorage: { removeItem() {} } }
)
function renderQuote(lines) {
	cursor = 0
	const result = runCheckoutQuote({ ...request, lines, menu: [] }, true, undefined, 'web', false)
	for (const effect of effects.splice(0)) effect()
	return result
}
renderQuote(request.lines)
renderQuote(request.lines)
renderQuote([line(1)])
assert.equal(requestsInFlight.length, 2)
requestsInFlight[1](validQuote)
await new Promise(setImmediate)
assert.equal(renderQuote([line(1)]).ready, true)
requestsInFlight[0](refusedQuote)
await new Promise(setImmediate)
const settled = renderQuote([line(1)])
assert.equal(settled.ready, true)
assert.equal(settled.unavailableLines, undefined)

products[1].isHidden = false
menu = await getMenu()
assert.deepEqual(ids(menu.dishes), ['1', '2', '4', '501'], 'Clearing visibility restores the dish')
assert.equal(menu.dishes.find((dish) => dish.id === '2').orderBlocked, true, 'Manual block is preserved')
products.forEach((dish) => {
	dish.isHidden = true
})
menu = await getMenu()
assert.equal(menu.dishes.length, 0)
assert.equal(priceLines([{ dishId: '2', variantId: 'default', quantity: 1 }], menu.dishes).count, 0)
assert.equal(
	selectHomepageMenu(menu, { categories: [{ categoryId: 1, productIds: [2] }] }, '/images/placeholder.svg').length,
	0
)
console.log(
	'Dish visibility: pagination, public visibility, cart availability, quote recovery and stale responses pass.'
)
