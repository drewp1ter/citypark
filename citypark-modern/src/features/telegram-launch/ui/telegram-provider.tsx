'use client'

import Script from 'next/script'
import { usePathname, useRouter } from 'next/navigation'
import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { launchTelegram } from '../api/actions'
import type { TelegramWebApp } from '../model/types'

type Status = 'loading' | 'unavailable' | 'invalid' | 'ready'
const insets = ['top', 'right', 'bottom', 'left'] as const

function updateViewport(app: TelegramWebApp) {
	const root = document.documentElement
	const heights = [app.viewportHeight, window.visualViewport?.height, window.innerHeight].filter(
		(value): value is number => typeof value === 'number' && value > 0
	)
	const height = Math.min(...heights)
	root.style.setProperty('--mini-height', `${Math.max(0, height)}px`)
	for (const side of insets) {
		const safe = app.safeAreaInset?.[side] ?? 0
		const content = app.contentSafeAreaInset?.[side] ?? 0
		root.style.setProperty(`--mini-${side}`, `${Math.max(0, safe + content)}px`)
	}
}

export function TelegramProvider({ authenticated, children }: { authenticated: boolean; children: ReactNode }) {
	const router = useRouter()
	const pathname = usePathname()
	const [status, setStatus] = useState<Status>('loading')
	const attempted = useRef(false)
	const appRef = useRef<TelegramWebApp | undefined>(undefined)

	const initialize = useCallback(async () => {
		const app = window.Telegram?.WebApp
		if (!app || !app.initData) {
			setStatus('unavailable')
			return
		}
		appRef.current = app
		updateViewport(app)
		app.expand?.()
		app.ready()
		if (authenticated) {
			setStatus('ready')
			return
		}
		if (attempted.current) return
		attempted.current = true
		// A lost response must not cause the same one-time Telegram proof to be posted again on reload.
		try {
			const bytes = new TextEncoder().encode(app.initData)
			const digest = await crypto.subtle.digest('SHA-256', bytes)
			const fingerprint = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('')
			const key = 'citypark-mini-launch'
			if (sessionStorage.getItem(key) === fingerprint) {
				setStatus('invalid')
				return
			}
			sessionStorage.setItem(key, fingerprint)
		} catch {
			setStatus('invalid')
			return
		}
		const result = await launchTelegram(app.initData)
		if (result === 'ok') {
			setStatus('ready')
			router.refresh()
		} else setStatus('invalid')
	}, [authenticated, router])

	useEffect(() => {
		// Script may already be present after client navigation.
		if (window.Telegram?.WebApp) queueMicrotask(() => void initialize())
		const timer = window.setTimeout(() => {
			if (!appRef.current) setStatus('unavailable')
		}, 5000)
		return () => window.clearTimeout(timer)
	}, [initialize])

	useEffect(() => {
		const app = appRef.current
		if (!app) return
		const update = () => updateViewport(app)
		const back = () => router.push('/mini')
		for (const name of ['viewportChanged', 'safeAreaChanged', 'contentSafeAreaChanged']) app.onEvent?.(name, update)
		window.visualViewport?.addEventListener('resize', update)
		if (pathname === '/mini') app.BackButton?.hide()
		else {
			app.BackButton?.show()
			app.BackButton?.onClick(back)
		}
		return () => {
			for (const name of ['viewportChanged', 'safeAreaChanged', 'contentSafeAreaChanged']) app.offEvent?.(name, update)
			window.visualViewport?.removeEventListener('resize', update)
			app.BackButton?.offClick(back)
			app.BackButton?.hide()
			for (const side of insets) document.documentElement.style.removeProperty(`--mini-${side}`)
			document.documentElement.style.removeProperty('--mini-height')
		}
	}, [pathname, router, status])

	return (
		<>
			<Script
				src='https://telegram.org/js/telegram-web-app.js'
				strategy='afterInteractive'
				onReady={() => void initialize()}
				onError={() => setStatus('unavailable')}
			/>
			{status === 'ready' && authenticated ? (
				children
			) : (
				<main
					id='mini-content'
					className='mx-auto flex min-h-[var(--mini-height,100dvh)] max-w-md items-center justify-center px-5 text-center'
				>
					<p role='status'>
						{status === 'invalid'
							? 'Не удалось войти. Закройте и снова откройте приложение'
							: status === 'unavailable'
								? 'Откройте приложение через Telegram'
								: 'Входим через Telegram…'}
					</p>
				</main>
			)}
		</>
	)
}
