// Isolated menu integration regression; no requests to external services.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'

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
assert.deepEqual(ids(cart.items.map((line) => line.dish)), ['1', '4', '501'])
assert.equal(cart.subtotal, 600)
assert.equal(cart.count, 6)

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
console.log('Dish visibility: pagination, categories, homepage, ID lookups, stale carts and restoration pass.')
