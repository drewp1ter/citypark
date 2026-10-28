export type WorkingHoursForm = { start: string; end: string }
export type WorkingHoursErrors = Partial<Record<keyof WorkingHoursForm, string>>

const REQUIRED = 'Укажите время'
const BAD_FORMAT = 'Время в формате ЧЧ:ММ, например 09:00'
const END_BEFORE_START = 'Время окончания должно быть позже начала'

export function toForm(h: { start: string; end: string }): WorkingHoursForm {
	return { start: h.start, end: h.end }
}

/** "9:00" → "09:00", "09.30" → "09:30"; anything that is not a time of day → null. */
export function normalizeTime(s: string): string | null {
	const match = /^(\d{1,2})[:.](\d{2})$/.exec(s.trim())
	if (!match) return null
	const hours = Number(match[1])
	const minutes = Number(match[2])
	if (hours > 23 || minutes > 59) return null
	return `${String(hours).padStart(2, '0')}:${match[2]}`
}

export function validateWorkingHours(f: WorkingHoursForm): WorkingHoursErrors {
	const errors: WorkingHoursErrors = {}
	const start = f.start.trim() === '' ? undefined : normalizeTime(f.start)
	const end = f.end.trim() === '' ? undefined : normalizeTime(f.end)
	if (f.start.trim() === '') errors.start = REQUIRED
	else if (start === null) errors.start = BAD_FORMAT
	if (f.end.trim() === '') errors.end = REQUIRED
	else if (end === null) errors.end = BAD_FORMAT
	// Both are "HH:MM", so comparing the strings compares the times.
	if (start && end && start >= end) errors.end = END_BEFORE_START
	return errors
}

/** Call after `validateWorkingHours` passes. */
export function toRequest(f: WorkingHoursForm): { start: string; end: string } {
	return { start: normalizeTime(f.start) ?? f.start, end: normalizeTime(f.end) ?? f.end }
}

export function mapServerFields(fields: Record<string, string> | undefined): WorkingHoursErrors {
	const errors: WorkingHoursErrors = {}
	if (fields?.start !== undefined) errors.start = fields.start
	if (fields?.end !== undefined) errors.end = fields.end
	return errors
}
