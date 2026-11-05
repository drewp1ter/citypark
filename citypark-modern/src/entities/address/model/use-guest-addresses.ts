'use client'

import { useSyncExternalStore } from 'react'
import { getGuestSnapshot, getServerGuestSnapshot, subscribeToGuestAddresses } from './guest-store'

export function useGuestAddresses() {
	return useSyncExternalStore(subscribeToGuestAddresses, getGuestSnapshot, getServerGuestSnapshot)
}
