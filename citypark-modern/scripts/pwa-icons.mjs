// Generates the PWA icons from app/icon.svg with Playwright Chromium. Run once and commit the PNGs.
// Outputs: public/pwa/icon-192.png, public/pwa/icon-512.png, public/pwa/icon-maskable-512.png, app/apple-icon.png.
// Usage: node scripts/pwa-icons.mjs
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'

const svg = readFileSync(new URL('../app/icon.svg', import.meta.url), 'utf8')
const browser = await chromium.launch()

async function shoot(html, size, file) {
	const page = await browser.newPage({ viewport: { width: size, height: size } })
	await page.setContent(`<style>html,body{margin:0}</style>${html}`)
	await page.screenshot({ path: fileURLToPath(new URL(file, import.meta.url)), omitBackground: true })
	await page.close()
	console.log('wrote', file)
}

const plain = (size) => `<div style="width:${size}px;height:${size}px">${svg.replace('<svg ', `<svg width="${size}" height="${size}" `)}</div>`

// Full-bleed lime square (the OS masks the corners); the glyph sits inside the central 80 % safe zone.
const full = (size) => `<div style="width:${size}px;height:${size}px;background:#D6F24A;display:flex;align-items:center;justify-content:center">
	<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="${size * 0.8}" height="${size * 0.8}">
		<text x="32" y="43" text-anchor="middle" font-family="Arial, Helvetica, sans-serif" font-size="30" font-weight="700" fill="#111111">СП</text>
	</svg></div>`

await shoot(plain(192), 192, '../public/pwa/icon-192.png')
await shoot(plain(512), 512, '../public/pwa/icon-512.png')
await shoot(full(512), 512, '../public/pwa/icon-maskable-512.png')
await shoot(full(180), 180, '../app/apple-icon.png')
await browser.close()
