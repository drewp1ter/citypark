'use client'

import { useEffect } from 'react'

// Production only: a worker in `next dev` would get in the way of HMR.
export function RegisterServiceWorker() {
	useEffect(() => {
		if (process.env.NODE_ENV !== 'production' || !('serviceWorker' in navigator)) return
		navigator.serviceWorker
			.register('/sw.js', { scope: '/', updateViaCache: 'none' })
			.catch((error) => console.error('Service worker registration failed', error))
	}, [])

	return null
}
