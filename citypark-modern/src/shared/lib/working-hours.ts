export type WorkingHours = { start: string; end: string } // "09:00", "22:00"

/** Used when the backend does not answer; the same value the backend starts with. */
export const defaultWorkingHours: WorkingHours = { start: '09:00', end: '22:00' }

/** "09:00" → "9:00". */
export function displayTime(hhmm: string): string {
	return hhmm.replace(/^0/, '')
}

/** "9:00–22:00" (en dash), the text shown on the pages. */
export function formatHours(h: WorkingHours): string {
	return `${displayTime(h.start)}–${displayTime(h.end)}`
}

/** "Мы работаем с 9:00 до 22:00": the same text the backend sends in errors.deliveryTime. */
export function hoursError(h: WorkingHours): string {
	return `Мы работаем с ${displayTime(h.start)} до ${displayTime(h.end)}`
}
