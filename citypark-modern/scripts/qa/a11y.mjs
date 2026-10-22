// Accessibility audit with axe-core: every page, light and dark, mobile and desktop.
// Usage: node scripts/qa/a11y.mjs [route ...]   (server must run on BASE_URL, default http://localhost:3111)
import AxeBuilder from '@axe-core/playwright'
import { chromium } from 'playwright'
import { seedCart } from './menu-cart.mjs'

const base = process.env.BASE_URL ?? 'http://localhost:3111'
const routes = process.argv.slice(2).length
	? process.argv.slice(2)
	: [
			'/',
			'/restaurant',
			'/rooms',
			'/hotel',
			'/menu',
			'/checkout',
			'/checkout/done',
			'/news',
			'/news/frittata',
			'/contacts',
			'/documents',
			'/nope',
		]
const sizes = { mobile: { width: 390, height: 844 }, desktop: { width: 1440, height: 900 } }
const cart = await seedCart([1, 2])

const browser = await chromium.launch()
const found = new Map() // rule id -> { impact, help, cases: Set, nodes: Map(html -> count) }
let checked = 0

for (const route of routes) {
	for (const [sizeName, viewport] of Object.entries(sizes)) {
		for (const theme of ['light', 'dark']) {
			const context = await browser.newContext({ viewport, colorScheme: theme })
			await context.addInitScript(
				([t, c]) => {
					localStorage.setItem('theme', t)
					localStorage.setItem('citypark-cart', JSON.stringify(c))
				},
				[theme, cart]
			)
			const page = await context.newPage()
			await page.goto(base + route, { waitUntil: 'load' })
			await page.waitForTimeout(500)
			const result = await new AxeBuilder({ page })
				.withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice'])
				.analyze()
			checked++
			for (const violation of result.violations) {
				const entry = found.get(violation.id) ?? {
					impact: violation.impact,
					help: violation.help,
					cases: new Set(),
					nodes: new Map(),
				}
				entry.cases.add(`${route} ${sizeName}/${theme}`)
				for (const node of violation.nodes) {
					const key = `${node.target.join(' ')} :: ${node.html.slice(0, 110)}`
					entry.nodes.set(key, (entry.nodes.get(key) ?? 0) + 1)
				}
				found.set(violation.id, entry)
			}
			await context.close()
		}
	}
}
await browser.close()

console.log(`axe-core checked ${checked} page states (${routes.length} routes x 2 sizes x 2 themes)\n`)
if (found.size === 0) console.log('No violations.')
for (const [id, entry] of found) {
	console.log(`[${entry.impact}] ${id}: ${entry.help}`)
	console.log(`   in ${entry.cases.size} states, e.g. ${[...entry.cases].slice(0, 4).join('; ')}`)
	for (const [node] of [...entry.nodes].slice(0, 4)) console.log(`   - ${node}`)
}
process.exitCode = found.size ? 1 : 0
