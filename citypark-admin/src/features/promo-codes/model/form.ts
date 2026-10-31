import type { components } from '../../../shared/api/schema'

export type Campaign = components['schemas']['PromoCampaign']
export type CampaignInput = components['schemas']['PromoCampaignInput']
export type CampaignForm = {
	code: string
	percentage: string
	enabled: boolean
	target: CampaignInput['target']
	productIds: number[]
	categoryIds: number[]
	minimumEligibleSubtotal: string
	maximumDiscount: string
	startsAt: string
	endsAt: string
}

/** Moscow has a fixed UTC+03:00 offset for the campaign dates supported here. */
export function toMoscowInput(value: string | null): string {
	if (!value) return ''
	const parts = new Intl.DateTimeFormat('sv-SE', {
		timeZone: 'Europe/Moscow',
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23',
	}).format(new Date(value))
	return parts.replace(' ', 'T')
}

export function fromMoscowInput(value: string): string | null {
	if (!value.trim()) return null
	if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) throw new Error('Укажите дату и время.')
	const date = new Date(value + ':00+03:00')
	if (!Number.isFinite(date.getTime()) || toMoscowInput(date.toISOString()) !== value)
		throw new Error('Укажите дату и время.')
	return value + ':00+03:00'
}

export function initialForm(campaign?: Campaign): CampaignForm {
	return {
		code: campaign?.code ?? '',
		percentage: String(campaign?.percentage ?? 10),
		enabled: campaign?.enabled ?? false,
		target: campaign?.target ?? 'all',
		productIds: campaign?.productIds ?? [],
		categoryIds: campaign?.categoryIds ?? [],
		minimumEligibleSubtotal: campaign?.minimumEligibleSubtotal?.toString() ?? '',
		maximumDiscount: campaign?.maximumDiscount?.toString() ?? '',
		startsAt: toMoscowInput(campaign?.startsAt ?? null),
		endsAt: toMoscowInput(campaign?.endsAt ?? null),
	}
}

export function formBody(form: CampaignForm): { body: CampaignInput; errors: Record<string, string> } {
	const errors: Record<string, string> = {}
	const code = form.code.trim().toUpperCase()
	if (!/^[A-Za-z0-9_-]{3,32}$/.test(form.code.trim()))
		errors.code = 'Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.'
	const percentage = Number(form.percentage)
	if (!form.percentage.trim() || !Number.isInteger(percentage) || percentage < 1 || percentage > 99)
		errors.percentage = 'Укажите целый процент от 1 до 99.'
	function money(field: 'minimumEligibleSubtotal' | 'maximumDiscount') {
		if (!form[field].trim()) return null
		const value = Number(form[field])
		if (!Number.isFinite(value) || value <= 0) errors[field] = 'Укажите положительную сумму.'
		return value
	}
	function date(field: 'startsAt' | 'endsAt') {
		try {
			return fromMoscowInput(form[field])
		} catch {
			errors[field] = 'Укажите дату и время.'
			return null
		}
	}
	const startsAt = date('startsAt'),
		endsAt = date('endsAt')
	if (startsAt && endsAt && endsAt <= startsAt) errors.endsAt = 'Дата окончания должна быть позже даты начала.'
	const productIds = form.target === 'all' ? [] : [...new Set(form.productIds)].sort((a, b) => a - b)
	const categoryIds = form.target === 'all' ? [] : [...new Set(form.categoryIds)].sort((a, b) => a - b)
	if (form.target === 'selected' && !productIds.length && !categoryIds.length)
		errors.target = 'Выберите все блюда или укажите блюда и категории.'
	const body: CampaignInput = {
		code,
		percentage,
		enabled: form.enabled,
		target: form.target,
		productIds,
		categoryIds,
		minimumEligibleSubtotal: money('minimumEligibleSubtotal'),
		maximumDiscount: money('maximumDiscount'),
		startsAt,
		endsAt,
	}
	return { body, errors }
}

export function parseCampaignSearch(search: Record<string, unknown>): { edit?: number | 'new' } {
	if (search.edit === 'new') return { edit: 'new' }
	const id = Number(search.edit)
	return Number.isSafeInteger(id) && id > 0 ? { edit: id } : {}
}
