import { describe, expect, it } from 'vitest'
import { normalizeHtml, textLength } from './rich-text'

describe('normalizeHtml', () => {
	it('turns empty markup into an empty string', () => {
		expect(normalizeHtml('<p></p>')).toBe('')
		expect(normalizeHtml('<p><br></p>')).toBe('')
		expect(normalizeHtml('  \n')).toBe('')
		expect(normalizeHtml('')).toBe('')
	})
	it('keeps real content unchanged', () => {
		expect(normalizeHtml('<p>a</p>')).toBe('<p>a</p>')
		expect(normalizeHtml('<ul><li></li></ul>')).toBe('<ul><li></li></ul>')
	})
})

describe('textLength', () => {
	it('decodes entities and counts one character for &nbsp;', () => {
		expect(textLength('<p>При&nbsp;вет</p>')).toBe(7)
		expect(textLength('<p>a &amp; b</p>')).toBe(5)
	})
	it('strips tags', () => {
		expect(textLength('<ul><li>ab</li></ul>')).toBe(2)
		expect(textLength('')).toBe(0)
	})
	it('counts characters outside the BMP once', () => {
		expect(textLength('<p>&#128512;</p>')).toBe(1)
	})
})
