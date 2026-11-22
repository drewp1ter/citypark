'use client'

import { useSyncExternalStore } from 'react'
import { todayInMoscow } from './dates'

const subscribeNever = () => () => {}

/**
 * Today's date in the hotel's Moscow time zone, e.g. "2026-10-16". It is "" on the server and during hydration,
 * so the pages can stay static (a date computed at build time would be wrong) and there is no hydration mismatch.
 */
export function useToday(): string {
	return useSyncExternalStore(
		subscribeNever,
		() => todayInMoscow(),
		() => ''
	)
}
