import { cache } from 'react'
import { serverEnv } from '@/shared/config'
import { defaultWorkingHours, type WorkingHours } from '@/shared/lib'

const hhmm = /^([01]\d|2[0-3]):[0-5]\d$/

/** GET /working-hours (revalidate 60); on any error logs it and returns `defaultWorkingHours`, so pages keep rendering. */
export const getWorkingHours = cache(async (): Promise<WorkingHours> => {
	try {
		const response = await fetch(serverEnv().MENU_API_URL + '/working-hours', { next: { revalidate: 60 } })
		if (!response.ok) throw new Error(`GET /working-hours: ${response.status}`)
		const { start, end } = (await response.json()) as { start?: unknown; end?: unknown }
		if (typeof start !== 'string' || typeof end !== 'string' || !hhmm.test(start) || !hhmm.test(end)) {
			throw new Error(`GET /working-hours: unexpected value ${String(start)}–${String(end)}`)
		}
		return { start, end }
	} catch (error) {
		console.error('Working hours: using the built-in hours.', error)
		return defaultWorkingHours
	}
})
