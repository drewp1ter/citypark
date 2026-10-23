import { describe, expect, it } from 'vitest'
import { emptyNews, mapServerFields, toForm, toRequest, validateNews, type NewsForm } from './form'

const valid: NewsForm = { ...emptyNews('2026-10-22'), title: 'Новые стейки' }

describe('emptyNews', () => {
	it('starts with today and no slug', () => {
		expect(emptyNews('2026-10-22')).toMatchObject({ date: '2026-10-22', slug: '', title: '', body: '' })
	})
})

describe('validateNews', () => {
	it('accepts a minimal form', () => {
		expect(validateNews(valid)).toEqual({})
	})

	it('requires a title', () => {
		expect(validateNews({ ...valid, title: '   ' }).title).toBe('Укажите заголовок')
	})

	it('limits title, short title and excerpt', () => {
		expect(validateNews({ ...valid, title: 'а'.repeat(200) }).title).toBeUndefined()
		expect(validateNews({ ...valid, title: 'а'.repeat(201) }).title).toBeDefined()
		expect(validateNews({ ...valid, shortTitle: 'а'.repeat(101) }).shortTitle).toBeDefined()
		expect(validateNews({ ...valid, excerpt: 'а'.repeat(301) }).excerpt).toBeDefined()
	})

	it.each(['novye-steyki', 'a', 'news-2', '2026'])('accepts the slug %j', (slug) => {
		expect(validateNews({ ...valid, slug }).slug).toBeUndefined()
	})

	it.each(['Novye', 'новости', 'a--b', '-a', 'a-', 'a b', 'a_b'])('rejects the slug %j', (slug) => {
		expect(validateNews({ ...valid, slug }).slug).toBe('Только латиница, цифры и дефисы')
	})

	it('allows an empty slug', () => {
		expect(validateNews({ ...valid, slug: '  ' }).slug).toBeUndefined()
	})

	it('limits the slug to 100 characters', () => {
		expect(validateNews({ ...valid, slug: 'a'.repeat(101) }).slug).toBeDefined()
	})

	it('requires a date', () => {
		expect(validateNews({ ...valid, date: '' }).date).toBeDefined()
	})

	it('counts only visible text of the body', () => {
		const tags = `<p>${'<strong></strong>'.repeat(5000)}${'а'.repeat(50_000)}</p>`
		expect(validateNews({ ...valid, body: tags }).body).toBeUndefined()
		expect(validateNews({ ...valid, body: `<p>${'а'.repeat(50_001)}</p>` }).body).toBeDefined()
	})

	it('checks the alt text only when there is a photo', () => {
		const long = 'а'.repeat(201)
		expect(validateNews({ ...valid, photo: { url: '', alt: long, position: '' } }).photo).toBeUndefined()
		expect(
			validateNews({ ...valid, photo: { url: '/uploads/a.jpg', alt: long, position: '' } }).photo?.alt
		).toBeDefined()
	})
})

describe('toRequest', () => {
	it('trims the texts and keeps an empty slug as an empty string', () => {
		const request = toRequest({
			...valid,
			title: '  Новые стейки ',
			shortTitle: ' Стейки ',
			excerpt: ' Три ',
			slug: '  ',
		})
		expect(request).toMatchObject({ title: 'Новые стейки', shortTitle: 'Стейки', excerpt: 'Три', slug: '' })
	})

	it('keeps the date as YYYY-MM-DD', () => {
		expect(toRequest(valid).date).toBe('2026-10-22')
	})
})

describe('toForm', () => {
	it('drops a photo position that is not "X% Y%"', () => {
		const item = { ...valid, photo: { url: '/images/a.jpg', alt: '', position: 'right center' } }
		expect(toForm(item).photo.position).toBe('')
		expect(toForm({ ...item, photo: { ...item.photo, position: '50% 30%' } }).photo.position).toBe('50% 30%')
	})
})

describe('mapServerFields', () => {
	it('maps text and photo keys and drops unknown ones', () => {
		expect(mapServerFields({ title: 'a', slug: 'b', 'photo.url': 'c', 'photo.position': 'd', other: 'e' })).toEqual({
			title: 'a',
			slug: 'b',
			photo: { url: 'c', position: 'd' },
		})
	})
})
