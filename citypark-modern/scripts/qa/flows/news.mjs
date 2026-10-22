// Flow check: news. Drives the real pages in a browser and prints what happened next to what is expected.
// Usage: node scripts/qa/flows/news.mjs   (server must run on BASE_URL, default http://localhost:3111)
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

let p = await open('/news', D)
out('desktop: status / h1 / title', `${p.status} / ${await txt(p.locator('h1'))} / ${await p.title()}`)
out('nav: Новости is current', await txt(p.locator("nav[aria-label='Основное меню'] a[aria-current=page]")))
out('latest card: badge + title + excerpt', await txt(p.locator("#news a[href^='/news/']").first()))
out(
	'desktop: small cards / rows',
	`${(await p.locator("#news a[href^='/news/']:has(img):visible").count()) - 1} cards / ${await p.locator('#news ul:visible li').count()} rows`
)
out(
	'rows (titles)',
	JSON.stringify(
		await p.locator('#news ul:visible li a').evaluateAll((a) => a.map((x) => x.querySelectorAll('span')[1].textContent))
	)
)
out(
	'social buttons (new tab, real hrefs)',
	JSON.stringify(
		await p
			.locator('main > section:last-of-type a')
			.evaluateAll((a) =>
				a.map((x) => `${x.textContent.trim()}→${x.getAttribute('href')} [${x.getAttribute('target')}]`)
			)
	)
)
out(
	'link targets: all news links -> articles',
	JSON.stringify(
		[...new Set(await p.locator('#news a').evaluateAll((a) => a.map((x) => x.getAttribute('href'))))].length
	) + ' unique'
)

out('tabs: none', String(await p.locator("nav[aria-label='Раздел']").count()))
out('blog: no #blog section', String(await p.locator('#blog').count()))

// article page from a card
await p.locator("#news a[href^='/news/']").first().click()
await p.waitForURL('**/news/**')
out('latest card opens article', `${p.url().replace(base, '')} / ${await txt(p.locator('h1'))}`)
out(
	'desktop: overflow / errors',
	`${await p.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(p.problems)}`
)

// mobile
const m = await open('/news', M)
out(
	'mobile: rows (5, short titles)',
	(await m.locator('#news ul:visible li').allInnerTexts()).map((t) => t.replace(/\s+/g, ' ')).join(' | ')
)
out('mobile: social CTA hidden', String(await m.getByText('Акции — первыми в соцсетях').isVisible()))
out(
	'mobile: quick bar',
	`${await txt(m.locator("nav[aria-label='Быстрые действия']"))} ${JSON.stringify(await m.locator("nav[aria-label='Быстрые действия'] a").evaluateAll((a) => a.map((x) => x.getAttribute('href'))))}`
)
out(
	'mobile: overflow / errors',
	`${await m.evaluate(() => document.documentElement.scrollWidth > innerWidth)} / ${JSON.stringify(m.problems)}`
)
await m.goto(base + '/news/special-offer-september-2026')
out('mobile: article page bar (also /news/…)', await txt(m.locator("nav[aria-label='Быстрые действия']")))

const gone = await open('/news/frittata', D)
out('/news/frittata (former blog slug) -> 404', `${gone.status}  (want 404)`)

for (const [name, v] of [
	['mobile', M],
	['desktop', D],
]) {
	const q = await open('/news', v)
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
			small: [...document.querySelectorAll('a,button')]
				.filter((e) => vis(e) && !e.classList.contains('tap-area') && size(e)[1] < 43.5)
				.map((e) => `${(e.getAttribute('aria-label') || e.textContent).trim().slice(0, 22)} ${size(e).join('x')}`),
		}
	})
	console.log(name.padEnd(8), JSON.stringify(info))
}
await browser.close()
