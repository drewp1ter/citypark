import { isOrderStatus, isPaymentType, type OrderStatus, type PaymentType } from './labels'

export const PAGE_SIZE = 20

export type OrderSearch = {
	q?: string
	from?: string
	to?: string
	status?: OrderStatus
	paymentType?: PaymentType
	page?: number
}

const dayPattern = /^(\d{4})-(\d{2})-(\d{2})$/

function parseDay(value: unknown): string | undefined {
	if (typeof value !== 'string') return undefined
	const match = dayPattern.exec(value)
	if (match === null) return undefined
	const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])]
	const date = new Date(Date.UTC(year, month - 1, day))
	const real = date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day
	return real ? value : undefined
}

function parsePage(value: unknown): number | undefined {
	const page = typeof value === 'string' && value.trim() !== '' ? Number(value) : value
	return typeof page === 'number' && Number.isInteger(page) && page >= 1 ? page : undefined
}

function parseQuery(value: unknown): string | undefined {
	// The router turns "123" into the number 123.
	const text = typeof value === 'number' ? String(value) : value
	if (typeof text !== 'string') return undefined
	const q = text.trim()
	return q !== '' && q.length <= 100 ? q : undefined
}

/** Reads the URL search params; every invalid value is dropped silently. */
export function parseOrderSearch(raw: Record<string, unknown>): OrderSearch {
	const from = parseDay(raw.from)
	let to = parseDay(raw.to)
	if (from !== undefined && to !== undefined && to < from) to = undefined
	const search: OrderSearch = {}
	const q = parseQuery(raw.q)
	if (q !== undefined) search.q = q
	if (from !== undefined) search.from = from
	if (to !== undefined) search.to = to
	if (isOrderStatus(raw.status)) search.status = raw.status
	if (isPaymentType(raw.paymentType)) search.paymentType = raw.paymentType
	const page = parsePage(raw.page)
	if (page !== undefined) search.page = page
	return search
}

/** Query string values for `GET /orders`; empty values are left out. */
export function toApiQuery(s: OrderSearch, pageSize: number): Record<string, string | number> {
	const query: Record<string, string | number> = { page: s.page ?? 1, pageSize }
	for (const key of ['q', 'from', 'to', 'status', 'paymentType'] as const) {
		const value = s[key]
		if (value !== undefined && value !== '') query[key] = value
	}
	return query
}

export function hasFilters(s: OrderSearch): boolean {
	return (
		s.q !== undefined ||
		s.from !== undefined ||
		s.to !== undefined ||
		s.status !== undefined ||
		s.paymentType !== undefined
	)
}
