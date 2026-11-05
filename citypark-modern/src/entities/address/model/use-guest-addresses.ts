'use client'

import { useSyncExternalStore } from 'react'
import { getGuestSnapshot, getServerGuestSnapshot, subscribeToGuestAddresses } from './guest-store'

const noop = () => () => {}

export function useGuestAddresses(enabled = true) {
	return useSyncExternalStore(
		enabled ? subscribeToGuestAddresses : noop,
		enabled ? getGuestSnapshot : getServerGuestSnapshot,
		getServerGuestSnapshot
	)
}
