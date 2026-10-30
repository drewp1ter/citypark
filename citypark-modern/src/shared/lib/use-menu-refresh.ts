'use client'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

export function useMenuRefresh() {
	const router = useRouter()
	useEffect(() => {
		const refresh = () => {
			if (document.visibilityState === 'visible') router.refresh()
		}
		refresh()
		window.addEventListener('focus', refresh)
		document.addEventListener('visibilitychange', refresh)
		const timer = window.setInterval(refresh, 60000)
		return () => {
			window.clearInterval(timer)
			window.removeEventListener('focus', refresh)
			document.removeEventListener('visibilitychange', refresh)
		}
	}, [router])
}
