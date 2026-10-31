'use client'

import { startTransition, useCallback, useEffect, useState } from 'react'
import { quoteOrder } from '../api/quote-order'
import type { OrderQuote } from '../api/orders-api'

type Request = { lines: unknown; method: string; city: string; menu: unknown }
type QuoteState = { key: string; result: Awaited<ReturnType<typeof quoteOrder>> }

/** Key checks invalidate pricing during render, before any effect or stale request can commit it. */
export function useCheckoutQuote(
	input: Request,
	enabled: boolean,
	refresh: number | undefined,
	surface: 'web' | 'mini',
	promoEnabled: boolean
) {
	const storageKey = `citypark-checkout-promo:${surface}`
	const [restored, setRestored] = useState(false)
	const [text, setText] = useState('')
	const [code, setCode] = useState('')
	const [editing, setEditing] = useState(false)
	const [revision, setRevision] = useState(0)
	const [state, setState] = useState<QuoteState>()
	const normalized = text.trim().replace(/[a-z]/g, (letter) => letter.toUpperCase())
	const dirty = promoEnabled && (editing || normalized !== code)
	useEffect(() => {
		let saved = ''
		try {
			if (promoEnabled) saved = sessionStorage.getItem(storageKey) ?? ''
			else sessionStorage.removeItem(storageKey)
		} catch {
			/* Storage can be unavailable. */
		}
		startTransition(() => {
			setText(saved)
			setCode(saved.trim().replace(/[a-z]/g, (letter) => letter.toUpperCase()))
			setEditing(false)
			setRestored(true)
		})
	}, [storageKey, promoEnabled])
	const remember = useCallback((value: string) => {
		try {
			if (value) sessionStorage.setItem(storageKey, value)
			else sessionStorage.removeItem(storageKey)
		} catch {
			/* Pricing still works without persistence. */
		}
	}, [storageKey])
	useEffect(() => {
		if (!promoEnabled || !editing) return
		const timer = window.setTimeout(() => {
			setCode(normalized)
			remember(normalized)
			setRevision((value) => value + 1)
			setEditing(false)
		}, 500)
		return () => window.clearTimeout(timer)
	}, [text, normalized, editing, promoEnabled, remember])
	const request = JSON.stringify({ lines: input.lines, method: input.method, city: input.city, promoCode: promoEnabled ? code : '' })
	const key = JSON.stringify([request, input.menu, revision, refresh])
	useEffect(() => {
		if (!enabled || !restored || dirty) return
		let current = true
		startTransition(async () => {
			const result = await quoteOrder(JSON.parse(request)).catch(() => ({
				ok: false as const,
				message: 'Не удалось рассчитать заказ. Попробуйте ещё раз.',
			}))
			if (current) setState({ key, result })
		})
		return () => {
			current = false
		}
	}, [request, key, enabled, dirty, restored])
	useEffect(() => {
		const refresh = () => {
			if (document.visibilityState === 'visible') setRevision((value) => value + 1)
		}
		window.addEventListener('focus', refresh)
		document.addEventListener('visibilitychange', refresh)
		const timer = window.setInterval(refresh, 60000)
		return () => {
			window.removeEventListener('focus', refresh)
			document.removeEventListener('visibilitychange', refresh)
			window.clearInterval(timer)
		}
	}, [])
	const result = enabled && restored && !dirty && state?.key === key ? state.result : undefined
	const quote: OrderQuote | undefined = result?.ok ? result.quote : undefined
	return {
		text,
		setText: (value: string) => {
			setText(value)
			if (value.trim() === '') {
				setCode('')
				setEditing(false)
				remember('')
				setRevision((value) => value + 1)
			} else setEditing(true)
		},
		quote,
		code: promoEnabled ? normalized : '',
		pending: enabled && (!restored || dirty || !result),
		ready: !!quote,
		promoError: result && !result.ok && 'fields' in result ? result.fields?.promoCode : undefined,
		error: result && !result.ok && !('fields' in result && result.fields?.promoCode) ? result.message : undefined,
		remove: () => {
			setText('')
			setCode('')
			setEditing(false)
			remember('')
			setRevision((value) => value + 1)
		},
	}
}
