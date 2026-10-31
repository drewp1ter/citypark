'use client'

import { startTransition, useEffect, useState } from 'react'
import { quoteOrder } from '../api/quote-order'
import type { OrderQuote } from '../api/orders-api'

type Request = { lines: unknown; method: string; city: string; menu: unknown }
type QuoteState = { key: string; result: Awaited<ReturnType<typeof quoteOrder>> }

/** Key checks invalidate pricing during render, before any effect or stale request can commit it. */
export function useCheckoutQuote(
	input: Request,
	enabled: boolean,
	refresh: number | undefined,
	surface: 'web' | 'mini'
) {
	const storageKey = `citypark-checkout-promo:${surface}`
	const [restored, setRestored] = useState(false)
	const [text, setText] = useState('')
	const [code, setCode] = useState('')
	const [revision, setRevision] = useState(0)
	const [state, setState] = useState<QuoteState>()
	const normalized = text.trim().replace(/[a-z]/g, (letter) => letter.toUpperCase())
	const dirty = normalized !== code
	useEffect(() => {
		let saved = ''
		try {
			saved = sessionStorage.getItem(storageKey) ?? ''
		} catch {
			/* Storage can be unavailable. */
		}
		startTransition(() => {
			setText(saved)
			setCode(saved)
			setRestored(true)
		})
	}, [storageKey])
	const remember = (value: string) => {
		try {
			if (value) sessionStorage.setItem(storageKey, value)
			else sessionStorage.removeItem(storageKey)
		} catch {
			/* Pricing still works without persistence. */
		}
	}
	const request = JSON.stringify({ lines: input.lines, method: input.method, city: input.city, promoCode: code })
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
		setText,
		quote,
		code: normalized,
		pending: enabled && !dirty && !result,
		ready: !!quote,
		error: dirty
			? 'Примените промокод или удалите его.'
			: result && !result.ok
				? ('fields' in result ? result.fields?.promoCode : undefined) || result.message
				: undefined,
		apply: () => {
			setCode(normalized)
			remember(normalized)
			setRevision((value) => value + 1)
		},
		remove: () => {
			setText('')
			setCode('')
			remember('')
			setRevision((value) => value + 1)
		},
	}
}
