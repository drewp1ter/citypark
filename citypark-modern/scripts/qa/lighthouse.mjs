// Lighthouse for one page, through a Playwright browser (Lighthouse cannot start Chromium itself in some sandboxes).
// Needs `npm i --no-save lighthouse` (it is not a project dependency).
// Usage: node scripts/qa/lighthouse.mjs <url> <output-name> [mobile|desktop]
// Scores on a shared machine vary by +-8 points between runs: compare medians of 5, not single runs.
import { chromium } from 'playwright'
import lighthouse from 'lighthouse'
import { writeFileSync } from 'node:fs'

const [url, name, mode = 'mobile'] = process.argv.slice(2)
const browser = await chromium.launch({ args: ['--remote-debugging-port=9333', '--no-sandbox'] })
const config =
	mode === 'desktop'
		? {
				extends: 'lighthouse:default',
				settings: {
					formFactor: 'desktop',
					screenEmulation: { mobile: false, width: 1350, height: 940, deviceScaleFactor: 1, disabled: false },
					throttlingMethod: 'simulate',
					throttling: {
						rttMs: 40,
						throughputKbps: 10240,
						cpuSlowdownMultiplier: 1,
						requestLatencyMs: 0,
						downloadThroughputKbps: 0,
						uploadThroughputKbps: 0,
					},
				},
			}
		: undefined
const result = await lighthouse(
	url,
	{
		port: 9333,
		output: 'json',
		logLevel: 'error',
		onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo'],
	},
	config
)
await browser.close()
const lhr = result.lhr
writeFileSync(`${name}.json`, JSON.stringify(lhr))
const score = (k) => Math.round(lhr.categories[k].score * 100)
const a = lhr.audits
console.log(
	`${name.padEnd(22)} perf ${score('performance')}  a11y ${score('accessibility')}  best-practices ${score('best-practices')}  seo ${score('seo')} | LCP ${a['largest-contentful-paint'].displayValue}  CLS ${a['cumulative-layout-shift'].displayValue}  TBT ${a['total-blocking-time'].displayValue}  FCP ${a['first-contentful-paint'].displayValue}`
)
for (const [id, audit] of Object.entries(a))
	if (
		audit.score !== null &&
		audit.score < 0.9 &&
		audit.scoreDisplayMode !== 'informative' &&
		audit.scoreDisplayMode !== 'notApplicable' &&
		audit.scoreDisplayMode !== 'manual' &&
		!['performance-budget', 'timing-budget'].includes(id)
	)
		console.log(`   ! ${id}: ${audit.title} (${Math.round(audit.score * 100)})`)
