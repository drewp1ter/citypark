import { afterEach, describe, expect, it, vi } from 'vitest'
import { imageSrc } from './image-src'

describe('imageSrc', () => {
	afterEach(() => vi.unstubAllEnvs())

	it('keeps an empty string empty', () => {
		expect(imageSrc('')).toBe('')
	})
	it('keeps absolute URLs', () => {
		expect(imageSrc('http://localhost:3000/uploads/2026/10/a.png')).toBe('http://localhost:3000/uploads/2026/10/a.png')
	})
	it('serves /images/ paths from the site', () => {
		vi.stubEnv('VITE_SITE_URL', 'https://site.example')
		expect(imageSrc('/images/room-vip.jpg')).toBe('https://site.example/images/room-vip.jpg')
	})
})
