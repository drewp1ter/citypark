// Flow check: restaurant. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/restaurant.mjs   (server must run on BASE_URL, default http://localhost:3111)
import { chromium } from 'playwright'
const base = process.env.BASE_URL ?? 'http://localhost:3111'
const browser = await chromium.launch()
const out = (k, v) => console.log(k.padEnd(56), v)
const txt = async (loc) =>
	(
		await loc
			.first()
			.innerText()
			.catch(() => '—')
	).replace(/\s+/g, ' ')
async function open(path, viewport) {
	const ctx = await browser.newContext({ viewport })
	const p = await ctx.newPage()
	const problems = []
	p.on('pageerror', (e) => problems.push(e.message.slice(0, 120)))
	const res = await p.goto(base + path)
	await p.waitForTimeout(500)
	return Object.assign(p, { problems, status: res.status() })
}
const D = { width: 1440, height: 900 },
	M = { width: 390, height: 844 }

let p = await open('/restaurant', D)
out('desktop: status / h1 / title', `${p.status} / ${await txt(p.locator('h1'))} / ${await p.title()}`)
out(
	'hero buttons',
	JSON.stringify(
		await p
			.locator('main section')
			.first()
			.locator('a:visible')
			.evaluateAll((a) => a.map((x) => `${x.textContent.trim().slice(0, 22)}→${x.getAttribute('href')}`).slice(2, 7))
	)
)
out(
	'facts (4)',
	JSON.stringify(
		await p
			.locator('main section')
			.first()
			.locator('ul li')
			.evaluateAll((l) => l.map((x) => x.innerText.replace(/\s+/g, ' ')))
	)
)
out(
	'sections: venues / banquets cards',
	`${await p.locator('#tour h2').allInnerTexts()} / ${(await p.locator("main section h2:text-is('Банкеты') >> xpath=../div/div/span[1]").allInnerTexts()).join(',')}`
)
out(
	'header: nav current + phone',
	`${await txt(p.locator("nav[aria-label='Основное меню'] a[aria-current=page]"))} / ${await p.getByRole('link', { name: 'Позвонить' }).first().getAttribute('href')}`
)

// no booking form: guests call
out(
	'booking anchor count / hero call link',
	`${await p.locator('[id=booking]').count()} / ${await p.locator('main section').first().getByRole('link', { name: 'Позвонить' }).getAttribute('href')}  (want 0 / tel:+79183119791)`
)
out(
	'desktop: overflow / errors',
	`${await p.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(p.problems)}`
)

// home: no form in the restaurant section
const h = await open('/', D)
out(
	'home: forms in the restaurant section',
	`${await h.locator('section:has(h2:text-is("Ресторан")) form').count()}  (want 0)`
)

// mobile
const m = await open('/restaurant', M)
out(
	'mobile: breadcrumbs / h1',
	`${await txt(m.locator("nav[aria-label='Хлебные крошки']:visible"))} / ${await txt(m.locator('h1'))}`
)
out(
	'mobile: photos in rail',
	String(await m.locator('main section').first().locator('div.overflow-x-auto img').count())
)
out('mobile: venue cards (short texts)', (await m.locator('#tour p:visible').allInnerTexts()).join(' | '))
out('mobile: banquets', await txt(m.locator('main section').nth(2).locator('div:visible').first()))
out(
	'mobile: quick bar',
	`${await txt(m.locator("nav[aria-label='Быстрые действия']"))} ${JSON.stringify(await m.locator("nav[aria-label='Быстрые действия'] a").evaluateAll((a) => a.map((x) => x.getAttribute('href'))))}`
)
out(
	"mobile: quick bar 'Меню' href",
	`${await m.locator("nav[aria-label='Быстрые действия'] a", { hasText: 'Меню' }).getAttribute('href')}  (want /menu)`
)
out(
	'mobile: overflow / errors',
	`${await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(m.problems)}`
)

for (const [name, v] of [
	['mobile', M],
	['desktop', D],
]) {
	const q = await open('/restaurant', v)
	const info = await q.evaluate(() => {
		const vis = (e) => e.offsetParent !== null || getComputedStyle(e).position === 'fixed'
		const size = (e) => {
			const r = e.getBoundingClientRect()
			return [Math.round(r.width), Math.round(r.height)]
		}
		return {
			h1: document.querySelectorAll('h1').length,
			headings: [...document.querySelectorAll('h1,h2,h3')]
				.filter(vis)
				.map((h) => h.tagName)
				.join(''),
			imgNoAlt: [...document.images].filter((i) => !i.hasAttribute('alt')).length,
			unlabelled: [...document.querySelectorAll('input:not([type=hidden]),select')].filter(
				(i) => vis(i) && !i.closest('[aria-hidden]') && !i.labels?.length
			).length,
			small: [...document.querySelectorAll('a,button,input:not([type=hidden]),select')]
				.filter((e) => vis(e) && !e.closest('[aria-hidden]') && !e.classList.contains('tap-area') && size(e)[1] < 43.5)
				.map(
					(e) => `${(e.getAttribute('aria-label') || e.textContent || e.name).trim().slice(0, 22)} ${size(e).join('x')}`
				),
		}
	})
	console.log(name.padEnd(8), JSON.stringify(info))
}
await browser.close()
