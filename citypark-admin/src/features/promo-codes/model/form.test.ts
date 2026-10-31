import { describe, expect, it } from 'vitest'
import { formBody, fromMoscowInput, initialForm, parseCampaignSearch, toMoscowInput, type Campaign } from './form'

describe('campaign form', () => {
	it('defaults to disabled with nullable optional fields and normalizes codes', () => {
		const { body, errors } = formBody({ ...initialForm(), code: ' autumn10 ' })
		expect(errors).toEqual({})
		expect(body).toEqual({
			code: 'AUTUMN10',
			percentage: 10,
			enabled: false,
			target: 'all',
			productIds: [],
			categoryIds: [],
			minimumEligibleSubtotal: null,
			maximumDiscount: null,
			startsAt: null,
			endsAt: null,
		})
	})
	it.each(['1', '99'])('accepts percentage %s', (percentage) => {
		expect(formBody({ ...initialForm(), code: 'ABC', percentage }).errors).toEqual({})
	})
	it.each(['', ' ', '0', '100', '1.5', 'NaN'])('rejects percentage %s', (percentage) => {
		expect(formBody({ ...initialForm(), percentage }).errors.percentage).toBeTruthy()
	})
	it.each(['ab', 'АБВ', 'ßßß', 'A B', 'A'.repeat(33)])('rejects malformed code %s', (code) => {
		expect(formBody({ ...initialForm(), code }).errors.code).toBeTruthy()
	})
	it('keeps mixed selection, deduplicates it, and clears selection for all dishes', () => {
		const form = {
			...initialForm(),
			code: 'ABC',
			target: 'selected' as const,
			productIds: [3, 1, 3],
			categoryIds: [2, 2],
		}
		expect(formBody(form).body).toMatchObject({ productIds: [1, 3], categoryIds: [2] })
		expect(formBody({ ...form, target: 'all' }).body).toMatchObject({ productIds: [], categoryIds: [] })
		expect(formBody({ ...form, productIds: [], categoryIds: [] }).errors.target).toBeTruthy()
		expect(formBody({ ...form, productIds: [] }).errors.target).toBeUndefined()
		expect(formBody({ ...form, categoryIds: [] }).errors.target).toBeUndefined()
	})
	it('maps valid limits and rejects zero, negative and nonfinite limits', () => {
		expect(
			formBody({ ...initialForm(), minimumEligibleSubtotal: '1200.50', maximumDiscount: '100' }).body
		).toMatchObject({ minimumEligibleSubtotal: 1200.5, maximumDiscount: 100 })
		for (const value of ['0', '-1', 'Infinity', 'bad']) {
			const { errors } = formBody({ ...initialForm(), minimumEligibleSubtotal: value, maximumDiscount: value })
			expect(errors.minimumEligibleSubtotal).toBeTruthy()
			expect(errors.maximumDiscount).toBeTruthy()
		}
	})
	it('maps existing campaigns including nulls and optional dates', () => {
		const campaign: Campaign = {
			...formBody({ ...initialForm(), code: 'ABC' }).body,
			id: 1,
			enabled: true,
			productIds: [2],
			categoryIds: [],
			minimumEligibleSubtotal: 100,
			maximumDiscount: null,
			startsAt: '2026-10-31T09:00:00Z',
			endsAt: null,
		}
		expect(initialForm(campaign)).toMatchObject({
			code: 'ABC',
			enabled: true,
			productIds: [2],
			minimumEligibleSubtotal: '100',
			maximumDiscount: '',
			startsAt: '2026-10-31T12:00',
			endsAt: '',
		})
	})
	it('converts Moscow time independently of browser timezone', () => {
		const previous = process.env.TZ
		process.env.TZ = 'America/Los_Angeles'
		try {
			expect(fromMoscowInput('2026-10-31T12:00')).toBe('2026-10-31T12:00:00+03:00')
			expect(toMoscowInput('2026-10-31T09:00:00Z')).toBe('2026-10-31T12:00')
			expect(fromMoscowInput('')).toBeNull()
			expect(toMoscowInput(null)).toBe('')
			expect(() => fromMoscowInput('2026-02-30T12:00')).toThrow()
			expect(() => fromMoscowInput('bad')).toThrow()
		} finally {
			if (previous === undefined) delete process.env.TZ
			else process.env.TZ = previous
		}
	})
	it('rejects invalid dates and requires end after start', () => {
		expect(formBody({ ...initialForm(), startsAt: 'bad' }).errors.startsAt).toBeTruthy()
		for (const endsAt of ['2026-10-31T11:00', '2026-10-31T12:00'])
			expect(formBody({ ...initialForm(), startsAt: '2026-10-31T12:00', endsAt }).errors.endsAt).toBeTruthy()
		expect(
			formBody({ ...initialForm(), startsAt: '2026-10-31T12:00', endsAt: '2026-11-01T12:00' }).errors.endsAt
		).toBeUndefined()
	})
	it('validates editor search params', () => {
		expect(parseCampaignSearch({ edit: 'new' })).toEqual({ edit: 'new' })
		expect(parseCampaignSearch({ edit: '12' })).toEqual({ edit: 12 })
		for (const edit of [undefined, '', -1, 0, 1.5, 'bad', Infinity]) expect(parseCampaignSearch({ edit })).toEqual({})
	})
})
