'use client'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

export function PendingPaymentRefresh() {
 const router = useRouter()
 useEffect(() => {
  let attempts = 0
  const timer = window.setInterval(() => {
   if (document.visibilityState !== 'visible') return
   if (++attempts >= 4) { window.clearInterval(timer); return }
   router.refresh()
  }, 3000)
  return () => window.clearInterval(timer)
 }, [router])
 return null
}
